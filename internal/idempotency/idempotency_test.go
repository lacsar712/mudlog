package idempotency_test

import (
	"errors"
	"testing"
	"time"

	"github.com/lacsar712/mudlog/internal/clock"
	"github.com/lacsar712/mudlog/internal/idempotency"
)

func TestRememberReplayAndConflict(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(0, 0))
	s := idempotency.New(clk, time.Hour)
	id, replay, err := s.Remember("key-aaaa", "hash1", "frm1")
	if err != nil || replay || id != "frm1" {
		t.Fatalf("first: %s replay=%v err=%v", id, replay, err)
	}
	id, replay, err = s.Remember("key-aaaa", "hash1", "frm2")
	if err != nil || !replay || id != "frm1" {
		t.Fatalf("replay: %s replay=%v err=%v", id, replay, err)
	}
	_, _, err = s.Remember("key-aaaa", "hash2", "frm3")
	if !errors.Is(err, idempotency.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	clk.Advance(2 * time.Hour)
	id, replay, err = s.Remember("key-aaaa", "hash2", "frm4")
	if err != nil || replay || id != "frm4" {
		t.Fatalf("after ttl: %s replay=%v err=%v", id, replay, err)
	}
}
