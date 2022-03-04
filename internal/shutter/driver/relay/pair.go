package relay

import (
	"context"
	"sync"
	"time"
)

func NewRelayPair(up, down Relay) (PairedRelay, PairedRelay) {
	l := &sync.Mutex{}

	return PairedRelay{l, up}, PairedRelay{l, down}
}

type PairedRelay struct {
	l *sync.Mutex
	r Relay
}

func (r *PairedRelay) EnableFor(ctx context.Context, duration time.Duration) error {
	r.l.Lock()
	defer r.l.Unlock()

	return r.r.EnableFor(ctx, duration)
}
