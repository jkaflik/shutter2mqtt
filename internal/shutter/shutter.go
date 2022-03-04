package shutter

import (
	"context"
)

const (
	ShutterOpenState    = "open"
	ShutterClosedState  = "closed"
	ShutterOpeningState = "opening"
	ShutterClosingState = "closing"
)

type ShutterUpdateHandler func(state string, position int)

type Shutter interface {
	Name() string
	FullOpenPosition() int
	FullClosePosition() int

	Position() int
	State() string

	OnUpdate(h ShutterUpdateHandler)

	Open(ctx context.Context) error
	Close(ctx context.Context) error
	Stop(ctx context.Context) error
	SetPosition(ctx context.Context, position int) error
}

type StatelessShutter interface {
	Shutter

	ResetPosition(position int) error
}
