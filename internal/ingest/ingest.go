package ingest

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lacsar712/mudlog/internal/clock"
	"github.com/lacsar712/mudlog/internal/fanout"
	"github.com/lacsar712/mudlog/internal/frame"
	"github.com/lacsar712/mudlog/internal/geology"
	"github.com/lacsar712/mudlog/internal/hashutil"
	"github.com/lacsar712/mudlog/internal/headers"
	"github.com/lacsar712/mudlog/internal/idempotency"
	"github.com/lacsar712/mudlog/internal/idgen"
	"github.com/lacsar712/mudlog/internal/job"
	"github.com/lacsar712/mudlog/internal/nonce"
	"github.com/lacsar712/mudlog/internal/queue"
	"github.com/lacsar712/mudlog/internal/sign"
	"github.com/lacsar712/mudlog/internal/wellkey"
)

type Pipeline struct {
	Clk    clock.Clock
	Window time.Duration
	Keys   *wellkey.Keys
	Nonces *nonce.Book
	Idem   *idempotency.Store
	Stores *geology.Registry
	Broker *queue.Broker
}

type Result struct {
	FrameID  string   `json:"frame_id"`
	Replay   bool     `json:"replay"`
	Matched  int      `json:"matched"`
	RelayIDs []string `json:"relay_ids"`
}

func (p *Pipeline) Handle(h http.Header, body []byte) (Result, int, error) {
	in, err := headers.ParseInbound(h)
	if err != nil {
		return Result{}, http.StatusBadRequest, err
	}
	if err := hashutil.ValidIdempotencyKey(in.IdemKey); err != nil {
		return Result{}, http.StatusBadRequest, err
	}
	secrets := p.Keys.Secrets(in.SourceKey)
	if len(secrets) == 0 {
		return Result{}, http.StatusUnauthorized, fmt.Errorf("unknown source key %q", in.SourceKey)
	}
	if err := sign.Verify(p.Clk, p.Window, secrets, sign.Headers{
		Timestamp: in.Timestamp,
		Nonce:     in.Nonce,
		Signature: in.Signature,
	}, body); err != nil {
		if errors.Is(err, sign.ErrSkew) {
			return Result{}, http.StatusBadRequest, err
		}
		return Result{}, http.StatusUnauthorized, err
	}
	env, err := frame.Parse(body)
	if err != nil {
		return Result{}, http.StatusUnprocessableEntity, err
	}
	if err := p.Nonces.CheckAndRemember(in.Nonce); err != nil {
		return Result{}, http.StatusConflict, err
	}
	now := p.Clk.Now()
	frameID := idgen.New("frm", now)
	bodyHash := hashutil.SHA256Hex(body)
	existing, replay, err := p.Idem.Remember(in.IdemKey, bodyHash, frameID)
	if err != nil {
		if errors.Is(err, idempotency.ErrConflict) {
			return Result{}, http.StatusConflict, err
		}
		return Result{}, http.StatusBadRequest, err
	}
	if replay {
		return Result{FrameID: existing, Replay: true}, http.StatusOK, nil
	}
	matched := p.Stores.Matching(env.Type)
	plan := fanout.PlanStores(frameID, env.Type, body, matched, now)
	ids := make([]string, 0, len(plan.Items))
	for _, item := range plan.Items {
		d, ok := p.Stores.Get(item.StoreID)
		if !ok {
			continue
		}
		p.Broker.Ensure(d.ID, d.Ordered, d.MaxInFlight)
		p.Broker.Enqueue(job.Job{
			FrameID:   frameID,
			RelayID:   item.RelayID,
			StoreID:   item.StoreID,
			Type:      env.Type,
			Body:      append([]byte(nil), body...),
			Attempt:   0,
			NotBefore: now,
			CreatedAt: now,
		}, d.Ordered, d.MaxInFlight)
		ids = append(ids, item.RelayID)
	}
	return Result{
		FrameID:  frameID,
		Matched:  len(ids),
		RelayIDs: ids,
	}, http.StatusAccepted, nil
}
