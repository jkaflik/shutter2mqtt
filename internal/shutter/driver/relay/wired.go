package relay

import (
	"context"
	"errors"
	"time"

	"github.com/racerxdl/go-mcp23017"
	"github.com/sirupsen/logrus"
)

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
	if !m.device.IsPresent() {
		return errors.New("device not alive")
	}

	logrus.Debugf("mcp23017: enable HIGH on %d", m.pin)

	return m.device.DigitalWrite(m.pin, mcp23017.LOW) // todo understand why LOW is HIGH and opposite
}

func (m *Mcp23017Pin) Low() error {
	if !m.device.IsPresent() {
		return errors.New("device not alive")
	}

	logrus.Debugf("mcp23017: enable LOW on %d", m.pin)

	return m.device.DigitalWrite(m.pin, mcp23017.HIGH) // todo understand why LOW is HIGH and opposite
}

type SetPin interface {
	High() error
	Low() error
}

type Wired struct {
	Pin          SetPin
	NormalClosed bool
}

func (p *Wired) EnableFor(ctx context.Context, duration time.Duration) error {
	after := time.After(duration)
	if err := p.enable(); err != nil {
		return err
	}
	defer func() {
		if err := p.disable(); err != nil {
			logrus.Error(err)
		}
	}()

	for {
		select {
		case <-after:
			return nil
		case <-ctx.Done():
			logrus.Debug("wired relay context exit")
			return ctx.Err()
		}
	}
}

func (p *Wired) enable() error {
	return p.Pin.High()
}

func (p *Wired) disable() error {
	return p.Pin.Low()
}
