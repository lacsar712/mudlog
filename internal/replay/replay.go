package replay

import (
	"fmt"
	"time"

	"github.com/lacsar712/mudlog/internal/dlq"
	"github.com/lacsar712/mudlog/internal/idgen"
	"github.com/lacsar712/mudlog/internal/job"
	"github.com/lacsar712/mudlog/internal/journal"
)

func FromJournal(e journal.Entry, now time.Time) (job.Job, error) {
	return job.Job{
		FrameID:   e.FrameID,
		RelayID:   idgen.New("rly", now),
		StoreID:   e.StoreID,
		Type:      e.Type,
		Body:      append([]byte(nil), e.Body...),
		Attempt:   0,
		NotBefore: now,
		CreatedAt: now,
		ReplayOf:  e.RelayID,
	}, nil
}

func FromDLQ(it dlq.Item, now time.Time) (job.Job, error) {
	if len(it.Body) == 0 {
		return job.Job{}, fmt.Errorf("dlq item %s has no stored body", it.RelayID)
	}
	return job.Job{
		FrameID:   it.FrameID,
		RelayID:   idgen.New("rly", now),
		StoreID:   it.StoreID,
		Type:      it.Type,
		Body:      append([]byte(nil), it.Body...),
		Attempt:   0,
		NotBefore: now,
		CreatedAt: now,
		ReplayOf:  it.RelayID,
	}, nil
}
