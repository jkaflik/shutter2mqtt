package main

import (
	"context"
	"github.com/racerxdl/go-mcp23017"
	"github.com/sirupsen/logrus"
	"time"
)

type Relay interface {
	EnableFor(ctx context.Context, duration time.Duration) error
}

type RelayPool struct {
	r Relay
	c chan struct{}
}

func (p *RelayPool) EnableFor(ctx context.Context, duration time.Duration) error {
	p.c <- struct{}{}
	defer func() {
		<-p.c
	}()

	return p.r.EnableFor(ctx, duration)
}

type DumbRelay struct {
	Name string
}

func (r *DumbRelay) EnableFor(ctx context.Context, duration time.Duration) error {
	t := time.After(duration)

	logrus.Debugf("%s: dumb relay start (for %s)", r.Name, duration.String())

	for {
		select {
		case <-t:
			logrus.Debugf("%s: dumb relay done", r.Name)
			return nil
		case <-ctx.Done():
			logrus.Debugf("%s: dumb relay exit", r.Name)
			return nil
		}
	}
}

type Mcp23017Pin struct {
	device *mcp23017.Device
	pin    uint8
}

func NewMcp23017Pin(device *mcp23017.Device, pin uint8) (p *Mcp23017Pin, err error) {
	p = &Mcp23017Pin{}
	p.device = device
	p.pin = pin
	err = p.device.PinMode(pin, mcp23017.OUTPUT)
	return p, err
}

func (m *Mcp23017Pin) High() error {
	return m.device.DigitalWrite(m.pin, mcp23017.HIGH)
}

func (m *Mcp23017Pin) Low() error {
	return m.device.DigitalWrite(m.pin, mcp23017.LOW)
}

type SetPin interface {
	High() error
	Low() error
}

type PinRelay struct {
	Pin         SetPin
	NormalClose bool
}

func (p *PinRelay) EnableFor(ctx context.Context, duration time.Duration) error {
	after := time.After(duration)
	if err := p.enable(); err != nil {
		return err
	}
	defer p.disable()

	for {
		select {
		case <-after:
		case <-ctx.Done():
			return nil
		}

		time.Sleep(time.Millisecond)
	}
}

func (p *PinRelay) enable() error {
	if !p.NormalClose {
		return p.Pin.High()
	}

	return p.Pin.Low()
}

func (p *PinRelay) disable() error {
	if !p.NormalClose {
		return p.Pin.Low()
	}

	return p.Pin.High()
}
