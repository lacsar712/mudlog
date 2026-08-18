package worker

import (
	"context"
	"log"
	"time"

	"github.com/lacsar712/mudlog/internal/backoff"
	"github.com/lacsar712/mudlog/internal/classify"
	"github.com/lacsar712/mudlog/internal/clock"
	"github.com/lacsar712/mudlog/internal/dlq"
	"github.com/lacsar712/mudlog/internal/geology"
	"github.com/lacsar712/mudlog/internal/job"
	"github.com/lacsar712/mudlog/internal/journal"
	"github.com/lacsar712/mudlog/internal/queue"
	"github.com/lacsar712/mudlog/internal/runtime"
	"github.com/lacsar712/mudlog/internal/wits"
)

type Engine struct {
	clk    clock.Clock
	broker *queue.Broker
	stores *geology.Registry
	gates  *runtime.Gates
	client *wits.Client
	log    *journal.Log
	dead   *dlq.Queue
	policy backoff.Policy
}

func New(clk clock.Clock, broker *queue.Broker, stores *geology.Registry, gates *runtime.Gates, client *wits.Client, jlog *journal.Log, dead *dlq.Queue, policy backoff.Policy) *Engine {
	return &Engine{
		clk:    clk,
		broker: broker,
		stores: stores,
		gates:  gates,
		client: client,
		log:    jlog,
		dead:   dead,
		policy: policy.Validate(),
	}
}

func (e *Engine) Run(ctx context.Context, n int) {
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		go e.loop(ctx)
	}
}

func (e *Engine) loop(ctx context.Context) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	j, ok := e.broker.Lease()
	if !ok {
		return
	}
	defer e.broker.Release(j.StoreID)
	e.handle(ctx, j)
}

func (e *Engine) handle(ctx context.Context, j job.Job) {
	dest, ok := e.stores.Get(j.StoreID)
	if !ok || !dest.Enabled {
		e.dead.Push(dlq.FromJob(j, "store missing or disabled", e.clk.Now()))
		e.note(j, "terminal", 0, "", "store_unavailable")
		return
	}
	e.gates.Ensure(dest.ID, dest.Rate, dest.Burst)
	br := e.gates.Breaker(dest.ID)
	dec := br.Allow()
	if !dec.Allow {
		j.NotBefore = e.clk.Now().Add(200 * time.Millisecond)
		e.broker.Enqueue(j, dest.Ordered, dest.MaxInFlight)
		e.note(j, "skipped_open", 0, "", dec.Note)
		return
	}
	wait := e.gates.Bucket(dest.ID).Take()
	if wait > 0 {
		j.NotBefore = e.clk.Now().Add(wait)
		e.broker.Enqueue(j, dest.Ordered, dest.MaxInFlight)
		e.note(j, "rate_delayed", 0, "", wait.String())
		return
	}

	j.Attempt++
	res := e.client.Post(context.Background(), wits.Request{
		URL:     dest.URL,
		Secret:  dest.Secret,
		FrameID: j.FrameID,
		RelayID: j.RelayID,
		StoreID: j.StoreID,
		Attempt: j.Attempt,
		Body:    j.Body,
		Now:     e.clk.Now(),
	})
	kind := classify.Combine(res.Status, res.Error)
	errMsg := ""
	if res.Error != nil {
		errMsg = res.Error.Error()
	}
	e.noteWithStatus(j, kind.String(), res.Status, errMsg, res.Body)

	switch kind {
	case classify.Success:
		br.Success()
		return
	case classify.Retryable:
		br.Failure()
		if e.policy.Exhausted(j.Attempt) {
			e.dead.Push(dlq.FromJob(j, "attempts exhausted", e.clk.Now()))
			e.note(j, "terminal", res.Status, errMsg, "exhausted")
			return
		}
		delay := e.policy.Delay(j.Attempt)
		j.NotBefore = e.clk.Now().Add(delay)
		e.broker.Enqueue(j, dest.Ordered, dest.MaxInFlight)
	case classify.Terminal:
		br.Failure()
		e.dead.Push(dlq.FromJob(j, "terminal http status", e.clk.Now()))
	default:
		log.Printf("unknown classify kind for %s", j.RelayID)
	}
}

func (e *Engine) note(j job.Job, kind string, status int, errMsg, note string) {
	e.noteWithStatus(j, kind, status, errMsg, note)
}

func (e *Engine) noteWithStatus(j job.Job, kind string, status int, errMsg, note string) {
	if len(note) > 240 {
		note = note[:240]
	}
	e.log.Append(journal.Entry{
		At:       e.clk.Now(),
		FrameID:  j.FrameID,
		RelayID:  j.RelayID,
		StoreID:  j.StoreID,
		Attempt:  j.Attempt,
		Kind:     kind,
		Status:   status,
		Error:    errMsg,
		Note:     note,
		Type:     j.Type,
		Body:     j.Body,
		ReplayOf: j.ReplayOf,
	})
}
