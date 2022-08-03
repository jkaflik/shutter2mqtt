package relay

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type Relay interface {
	EnableFor(ctx context.Context, duration time.Duration) error
	IsEnabled() bool
}

type PoolProxy struct {
	r Relay
	c chan struct{}
}

func NewPoolProxy(r Relay, pool chan struct{}) *PoolProxy {
	return &PoolProxy{r: r, c: pool}
}

func (p *PoolProxy) EnableFor(ctx context.Context, duration time.Duration) error {
	p.c <- struct{}{}
	defer func() {
		<-p.c
	}()

	return p.r.EnableFor(ctx, duration)
}

func (p *PoolProxy) IsEnabled() bool {
	return p.r.IsEnabled()
}

type Dumb struct {
	Name string

	isEnabled bool
}

func (r *Dumb) EnableFor(ctx context.Context, duration time.Duration) error {
	r.isEnabled = true
	defer func() { r.isEnabled = false }()

	t := time.After(duration)

	logrus.Warnf("%s: dumb shutter start (for %s)", r.Name, duration.String())

	for {
		select {
		case <-t:
			logrus.Warnf("%s: dumb shutter done", r.Name)
			return nil
		case <-ctx.Done():
			logrus.Warnf("%s: dumb shutter exit", r.Name)
			return ctx.Err()
		}
	}
}

func (r *Dumb) IsEnabled() bool {
	return r.isEnabled
}
