package fanout

import (
	"time"

	"github.com/lacsar712/mudlog/internal/geology"
	"github.com/lacsar712/mudlog/internal/idgen"
)

type PlanItem struct {
	StoreID string
	RelayID string
	URL     string
	Ordered bool
}

type Plan struct {
	FrameID    string
	Type       string
	Items      []PlanItem
	DroppedOff int
}

func PlanStores(frameID, frameType string, body []byte, stores []geology.Store, now time.Time) Plan {
	_ = body
	p := Plan{FrameID: frameID, Type: frameType, Items: make([]PlanItem, 0, len(stores))}
	for _, d := range stores {
		if !d.Matches(frameType) {
			p.DroppedOff++
			continue
		}
		p.Items = append(p.Items, PlanItem{
			StoreID: d.ID,
			RelayID: idgen.New("rly", now),
			URL:     d.URL,
			Ordered: d.Ordered,
		})
	}
	return p
}
