package relay

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type Relay interface {
	EnableFor(ctx context.Context, duration time.Duration) error
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

type Dumb struct {
	Name string
}

func (r *Dumb) EnableFor(ctx context.Context, duration time.Duration) error {
	t := time.After(duration)

	logrus.Debugf("%s: dumb shutter start (for %s)", r.Name, duration.String())

	for {
		select {
		case <-t:
			logrus.Debugf("%s: dumb shutter done", r.Name)
			return nil
		case <-ctx.Done():
			logrus.Debugf("%s: dumb shutter exit", r.Name)
			return nil
		}
	}
}
