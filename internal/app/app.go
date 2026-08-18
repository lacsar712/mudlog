package app

import (
	"context"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/lacsar712/mudlog/internal/backoff"
	"github.com/lacsar712/mudlog/internal/clock"
	"github.com/lacsar712/mudlog/internal/config"
	"github.com/lacsar712/mudlog/internal/cuttings"
	"github.com/lacsar712/mudlog/internal/dlq"
	"github.com/lacsar712/mudlog/internal/geology"
	"github.com/lacsar712/mudlog/internal/idempotency"
	"github.com/lacsar712/mudlog/internal/ingest"
	"github.com/lacsar712/mudlog/internal/journal"
	"github.com/lacsar712/mudlog/internal/nonce"
	"github.com/lacsar712/mudlog/internal/queue"
	"github.com/lacsar712/mudlog/internal/replay"
	"github.com/lacsar712/mudlog/internal/runtime"
	"github.com/lacsar712/mudlog/internal/store"
	"github.com/lacsar712/mudlog/internal/wellkey"
	"github.com/lacsar712/mudlog/internal/wits"
	"github.com/lacsar712/mudlog/internal/worker"
)

const Version = "0.1.0"

type App struct {
	Cfg     config.Config
	Clk     clock.Clock
	Stores  *geology.Registry
	Idem    *idempotency.Store
	Nonces  *nonce.Book
	Broker  *queue.Broker
	Keys    *wellkey.Keys
	Log     *journal.Log
	Dead    *dlq.Queue
	Gates   *runtime.Gates
	Sink    *cuttings.Sink
	Pipe    *ingest.Pipeline
	Engine  *worker.Engine
	Snap    *store.File
	Started time.Time
}

func New(cfg config.Config) (*App, error) {
	clk := clock.Real{}
	a := &App{
		Cfg:     cfg,
		Clk:     clk,
		Stores:  geology.NewRegistry(clk),
		Idem:    idempotency.New(clk, cfg.IdemTTL),
		Nonces:  nonce.New(clk, cfg.Window),
		Broker:  queue.NewBroker(clk),
		Keys:    wellkey.New("rig", cfg.IngestSecret),
		Log:     journal.New(500),
		Dead:    dlq.New(200),
		Gates:   runtime.NewGates(clk),
		Sink:    cuttings.New(50),
		Started: time.Now(),
	}
	a.Pipe = &ingest.Pipeline{
		Clk:    clk,
		Window: cfg.Window,
		Keys:   a.Keys,
		Nonces: a.Nonces,
		Idem:   a.Idem,
		Stores: a.Stores,
		Broker: a.Broker,
	}
	a.Engine = worker.New(clk, a.Broker, a.Stores, a.Gates, wits.New(10*time.Second), a.Log, a.Dead, backoff.Default())
	snap, err := store.New(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	a.Snap = snap
	if s, ok, err := snap.Load(); err != nil {
		return nil, err
	} else if ok {
		store.Apply(a.deps(), s)
	}
	if len(a.Stores.List()) == 0 {
		if err := a.seedCuttings(); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *App) deps() store.Deps {
	return store.Deps{
		Stores: a.Stores,
		Idem:   a.Idem,
		Nonces: a.Nonces,
		Log:    a.Log,
		Dead:   a.Dead,
		Broker: a.Broker,
		Keys:   a.Keys,
		Gates:  a.Gates,
	}
}

func (a *App) seedCuttings() error {
	raw, err := url.JoinPath(a.Cfg.PublicBase, a.Cfg.CuttingsPath)
	if err != nil {
		return err
	}
	enabled := true
	d, err := a.Stores.Create(geology.CreateInput{
		Name:         "cuttings-sink",
		URL:          raw,
		Secret:       "dev-geology-secret",
		TypePrefixes: []string{""},
		Ordered:      false,
		Rate:         20,
		Burst:        20,
		MaxInFlight:  4,
		Enabled:      &enabled,
	})
	if err != nil {
		return err
	}
	a.Broker.Ensure(d.ID, d.Ordered, d.MaxInFlight)
	return nil
}

func (a *App) StartWorkers(ctx context.Context) {
	a.Engine.Run(ctx, a.Cfg.Workers)
	go a.snapshotLoop(ctx)
}

func (a *App) snapshotLoop(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			a.save()
			return
		case <-t.C:
			a.save()
		}
	}
}

func (a *App) save() {
	if err := a.Snap.Save(store.Capture(a.deps())); err != nil {
		log.Printf("snapshot: %v", err)
	}
}

func (a *App) Replay(relayID string) (string, error) {
	now := a.Clk.Now()
	if it, ok := a.Dead.Get(relayID); ok {
		j, err := replay.FromDLQ(it, now)
		if err != nil {
			return "", err
		}
		d, ok := a.Stores.Get(j.StoreID)
		if !ok {
			return "", os.ErrNotExist
		}
		a.Broker.Ensure(d.ID, d.Ordered, d.MaxInFlight)
		a.Broker.Enqueue(j, d.Ordered, d.MaxInFlight)
		_, _ = a.Dead.Remove(relayID)
		return j.RelayID, nil
	}
	e, ok := a.Log.Get(relayID)
	if !ok {
		return "", os.ErrNotExist
	}
	j, err := replay.FromJournal(e, now)
	if err != nil {
		return "", err
	}
	d, ok := a.Stores.Get(j.StoreID)
	if !ok {
		return "", os.ErrNotExist
	}
	a.Broker.Ensure(d.ID, d.Ordered, d.MaxInFlight)
	a.Broker.Enqueue(j, d.Ordered, d.MaxInFlight)
	return j.RelayID, nil
}
