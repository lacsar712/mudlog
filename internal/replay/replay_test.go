package replay_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/lacsar712/mudlog/internal/journal"
	"github.com/lacsar712/mudlog/internal/redact"
	"github.com/lacsar712/mudlog/internal/replay"
)

func TestFromJournalUsesOriginalBody(t *testing.T) {
	body := []byte(`{"type":"lithology.cuttings","payload":{"password":"hunter2","wellbore":"W-12H"}}`)
	e := journal.Entry{
		FrameID:      "frm1",
		RelayID:      "rly1",
		StoreID:      "sto1",
		Type:         "lithology.cuttings",
		Body:         body,
		BodyRedacted: redact.JSON(body),
	}
	j, err := replay.FromJournal(e, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(j.Body, body) {
		t.Fatalf("replay body %s want original, not redacted %s", j.Body, e.BodyRedacted)
	}
	if bytes.Contains(j.Body, []byte("***")) {
		t.Fatal("replay used masked payload")
	}
}

func TestFromJournalRejectsEmptyBody(t *testing.T) {
	_, err := replay.FromJournal(journal.Entry{
		FrameID: "frm1",
		RelayID: "rly-empty",
		StoreID: "sto1",
		Type:    "lithology.cuttings",
	}, time.Unix(1, 0))
	if err == nil {
		t.Fatal("empty journal body must not become a replay job")
	}
}
