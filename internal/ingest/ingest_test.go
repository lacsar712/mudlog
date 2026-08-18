package ingest_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/lacsar712/mudlog/internal/clock"
	"github.com/lacsar712/mudlog/internal/geology"
	"github.com/lacsar712/mudlog/internal/hashutil"
	"github.com/lacsar712/mudlog/internal/headers"
	"github.com/lacsar712/mudlog/internal/idempotency"
	"github.com/lacsar712/mudlog/internal/ingest"
	"github.com/lacsar712/mudlog/internal/nonce"
	"github.com/lacsar712/mudlog/internal/queue"
	"github.com/lacsar712/mudlog/internal/sign"
	"github.com/lacsar712/mudlog/internal/wellkey"
)

const sampleBody = `{"type":"lithology.cuttings","wellbore":"W-12H","lithology":{"code":"SS","percent":80},"lag_depth":{"measured_m":1842.5},"payload":{"rop_m_hr":12}}`

func TestPipelineAcceptsSignedFrame(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	stores := geology.NewRegistry(clk)
	enabled := true
	_, err := stores.Create(geology.CreateInput{
		Name:         "sink",
		URL:          "http://127.0.0.1:8080/api/v1/cuttings",
		Secret:       "abcdefgh",
		TypePrefixes: []string{"lithology"},
		Enabled:      &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := &ingest.Pipeline{
		Clk:    clk,
		Window: 5 * time.Minute,
		Keys:   wellkey.New("rig", "supersecret"),
		Nonces: nonce.New(clk, 5*time.Minute),
		Idem:   idempotency.New(clk, time.Hour),
		Stores: stores,
		Broker: queue.NewBroker(clk),
	}
	body := []byte(sampleBody)
	n := "abcdefghijklmnop"
	ts := clk.Now().Unix()
	sig, err := sign.Sign("supersecret", ts, n, body)
	if err != nil {
		t.Fatal(err)
	}
	h := make(http.Header)
	h.Set(headers.Timestamp, "1700000000")
	h.Set(headers.Nonce, n)
	h.Set(headers.Signature, sig)
	h.Set(headers.Idempotency, "idemkey1")
	h.Set(headers.SourceKey, "rig")
	res, code, err := p.Handle(h, body)
	if err != nil {
		t.Fatal(err)
	}
	if code != http.StatusAccepted {
		t.Fatalf("code %d", code)
	}
	if res.Matched != 1 {
		t.Fatalf("matched %d hash %s", res.Matched, hashutil.SHA256Hex(body))
	}
}

func TestPipelineRejectsPartialTypePrefix(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	stores := geology.NewRegistry(clk)
	enabled := true
	_, err := stores.Create(geology.CreateInput{
		Name:         "too-short",
		URL:          "http://127.0.0.1:8080/api/v1/cuttings",
		Secret:       "abcdefgh",
		TypePrefixes: []string{"li"},
		Enabled:      &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := &ingest.Pipeline{
		Clk:    clk,
		Window: 5 * time.Minute,
		Keys:   wellkey.New("rig", "supersecret"),
		Nonces: nonce.New(clk, 5*time.Minute),
		Idem:   idempotency.New(clk, time.Hour),
		Stores: stores,
		Broker: queue.NewBroker(clk),
	}
	body := []byte(sampleBody)
	n := "qrstuvwxyzabcdef"
	ts := clk.Now().Unix()
	sig, err := sign.Sign("supersecret", ts, n, body)
	if err != nil {
		t.Fatal(err)
	}
	h := make(http.Header)
	h.Set(headers.Timestamp, "1700000000")
	h.Set(headers.Nonce, n)
	h.Set(headers.Signature, sig)
	h.Set(headers.Idempotency, "idemkey-partial")
	h.Set(headers.SourceKey, "rig")
	res, code, err := p.Handle(h, body)
	if err != nil {
		t.Fatal(err)
	}
	if code != http.StatusAccepted {
		t.Fatalf("code %d", code)
	}
	if res.Matched != 0 {
		t.Fatalf("prefix %q must not match %q, matched=%d ids=%v", "li", "lithology.cuttings", res.Matched, res.RelayIDs)
	}
}

func newPipeline(t *testing.T, clk *clock.Frozen) *ingest.Pipeline {
	t.Helper()
	stores := geology.NewRegistry(clk)
	enabled := true
	if _, err := stores.Create(geology.CreateInput{
		Name:         "sink",
		URL:          "http://127.0.0.1:8080/api/v1/cuttings",
		Secret:       "abcdefgh",
		TypePrefixes: []string{"lithology"},
		Enabled:      &enabled,
	}); err != nil {
		t.Fatal(err)
	}
	return &ingest.Pipeline{
		Clk:    clk,
		Window: 5 * time.Minute,
		Keys:   wellkey.New("rig", "supersecret"),
		Nonces: nonce.New(clk, 5*time.Minute),
		Idem:   idempotency.New(clk, time.Hour),
		Stores: stores,
		Broker: queue.NewBroker(clk),
	}
}

func signedHeaders(t *testing.T, body []byte, nonce, idem string, ts int64) http.Header {
	t.Helper()
	sig, err := sign.Sign("supersecret", ts, nonce, body)
	if err != nil {
		t.Fatal(err)
	}
	h := make(http.Header)
	h.Set(headers.Timestamp, strconv.FormatInt(ts, 10))
	h.Set(headers.Nonce, nonce)
	h.Set(headers.Signature, sig)
	h.Set(headers.Idempotency, idem)
	h.Set(headers.SourceKey, "rig")
	return h
}

func TestPipelineSkewIsBadRequest(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := []byte(sampleBody)
	ts := clk.Now().Add(-20 * time.Minute).Unix()
	_, code, err := p.Handle(signedHeaders(t, body, "skewnonce16chars", "idem-skew-01", ts), body)
	if err == nil {
		t.Fatal("expected skew error")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("code %d want 400", code)
	}
}

func TestPipelineIdempotencyConflict(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body1 := []byte(`{"type":"lithology.cuttings","payload":{"id":1}}`)
	body2 := []byte(`{"type":"lithology.cuttings","payload":{"id":2}}`)
	ts := clk.Now().Unix()
	_, code1, err := p.Handle(signedHeaders(t, body1, "idemnonce16charA", "same-idem-key", ts), body1)
	if err != nil || code1 != http.StatusAccepted {
		t.Fatalf("first: code=%d err=%v", code1, err)
	}
	_, code2, err := p.Handle(signedHeaders(t, body2, "idemnonce16charB", "same-idem-key", ts), body2)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if code2 != http.StatusConflict {
		t.Fatalf("code %d want 409", code2)
	}
}

func TestPipelineUnknownSourceKeyUnauthorized(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := []byte(sampleBody)
	h := signedHeaders(t, body, "unknownsrc16char", "idem-unknown-01", clk.Now().Unix())
	h.Set(headers.SourceKey, "no-such-key")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unknown source key panicked: %v", r)
		}
	}()
	_, code, err := p.Handle(h, body)
	if err == nil {
		t.Fatal("expected unauthorized")
	}
	if code != http.StatusUnauthorized {
		t.Fatalf("code %d want 401", code)
	}
}

func TestPipelineInvalidJSONIsBadRequest(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := []byte(`{`)
	_, code, err := p.Handle(signedHeaders(t, body, "jsonnonce16chars", "idem-json-01", clk.Now().Unix()), body)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("broken json want 400, got %d", code)
	}
}

func TestPipelineMissingPayloadUnprocessable(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := []byte(`{"type":"lithology.cuttings"}`)
	_, code, err := p.Handle(signedHeaders(t, body, "paylnonce16chars", "idem-payload-01", clk.Now().Unix()), body)
	if err == nil {
		t.Fatal("expected payload error")
	}
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("missing payload want 422, got %d", code)
	}
}

func TestPipelineDuplicateNonceConflict(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(1_700_000_000, 0))
	p := newPipeline(t, clk)
	body := []byte(sampleBody)
	ts := clk.Now().Unix()
	n := "dupnonce16charsx"
	_, code1, err := p.Handle(signedHeaders(t, body, n, "idem-nonce-a", ts), body)
	if err != nil || code1 != http.StatusAccepted {
		t.Fatalf("first: code=%d err=%v", code1, err)
	}
	_, code2, err := p.Handle(signedHeaders(t, body, n, "idem-nonce-b", ts), body)
	if err == nil {
		t.Fatal("expected nonce conflict")
	}
	if code2 != http.StatusConflict {
		t.Fatalf("duplicate nonce want 409, got %d", code2)
	}
}
