package app_test

import (
	"testing"
	"time"

	"github.com/lacsar712/mudlog/internal/app"
	"github.com/lacsar712/mudlog/internal/config"
	"github.com/lacsar712/mudlog/internal/journal"
)

func TestReplayEmptyJournalBodyFails(t *testing.T) {
	a, err := app.New(config.Config{
		Addr:         ":0",
		DataDir:      t.TempDir(),
		IngestSecret: "dev-rig-secret",
		Window:       5 * time.Minute,
		IdemTTL:      time.Hour,
		Workers:      1,
		PublicBase:   "http://127.0.0.1:8080",
		CuttingsPath: "/api/v1/cuttings",
	})
	if err != nil {
		t.Fatal(err)
	}
	stores := a.Stores.List()
	if len(stores) == 0 {
		t.Fatal("expected seeded store")
	}
	a.Log.Append(journal.Entry{
		FrameID: "frm-empty",
		RelayID: "rly-empty-body",
		StoreID: stores[0].ID,
		Type:    "lithology.cuttings",
	})
	if _, err := a.Replay("rly-empty-body"); err == nil {
		t.Fatal("replay of empty journal body must fail")
	}
	if a.Broker.Depth() != 0 {
		t.Fatalf("failed replay must not enqueue, depth=%d", a.Broker.Depth())
	}
}
