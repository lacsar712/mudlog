package journal_test

import (
	"testing"
	"time"

	"github.com/lacsar712/mudlog/internal/journal"
)

func TestGetPreservesBodyForReplay(t *testing.T) {
	log := journal.New(50)
	body := []byte(`{"type":"lithology.cuttings","payload":{"password":"hunter2"}}`)
	log.Append(journal.Entry{
		At:      time.Unix(1, 0),
		FrameID: "frm1",
		RelayID: "rly1",
		StoreID: "sto1",
		Attempt: 1,
		Kind:    "terminal",
		Type:    "lithology.cuttings",
		Body:    body,
	})
	got, ok := log.Get("rly1")
	if !ok {
		t.Fatal("missing entry")
	}
	if string(got.Body) != string(body) {
		t.Fatalf("Get stripped body: %q", got.Body)
	}
	listed := log.List("", 10)
	if len(listed) != 1 {
		t.Fatalf("list n=%d", len(listed))
	}
	if listed[0].Body != nil {
		t.Fatalf("List must hide raw body, got %q", listed[0].Body)
	}
}
