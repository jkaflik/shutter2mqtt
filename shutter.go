package main

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	shutterOpenState    = "open"
	shutterClosedState  = "closed"
	shutterOpeningState = "opening"
	shutterClosingState = "closing"
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

type RelaysShutter struct {
	rUp   Relay
	rDown Relay

	name              string
	fullOpenPosition  int
	fullClosePosition int
	timeToClose       time.Duration

	updateHandler  ShutterUpdateHandler
	updateInterval time.Duration

	currentState    string
	currentPosition int

	cancelCurrentContext context.CancelFunc
}

func (s *RelaysShutter) ResetPosition(position int) error {
	s.currentPosition = position
	if s.currentPosition == s.fullClosePosition {
		s.currentState = shutterClosedState
	}
	s.currentState = shutterOpenState

	return nil
}

func NewRelaysShutter(name string, up Relay, down Relay, fullOpenPosition int, fullClosePosition int, timeToClose time.Duration) *RelaysShutter {
	shutter := &RelaysShutter{rUp: up, rDown: down, name: name, fullOpenPosition: fullOpenPosition, fullClosePosition: fullClosePosition, timeToClose: timeToClose}
	shutter.currentState = shutterOpenState
	shutter.currentPosition = shutter.fullOpenPosition
	return shutter
}

func (s *RelaysShutter) retainContext(parent context.Context) (ctx context.Context) {
	if s.cancelCurrentContext != nil {
		logrus.Debugf("%s: found previous operation context, cancel", s.name)
		s.cancelCurrentContext()
	}

	ctx, s.cancelCurrentContext = context.WithCancel(parent)
	return ctx
}

func (s *RelaysShutter) Name() string {
	return s.name
}

func (s *RelaysShutter) Position() int {
	return s.currentPosition
}

func (s *RelaysShutter) State() string {
	return s.currentState
}

func (s *RelaysShutter) FullOpenPosition() int {
	return s.fullOpenPosition
}

func (s *RelaysShutter) FullClosePosition() int {
	return s.fullClosePosition
}

func (s *RelaysShutter) OnUpdate(h ShutterUpdateHandler) {
	s.updateHandler = h
}

func (s *RelaysShutter) Open(ctx context.Context) error {
	logrus.Infof("%s: open", s.name)
	ctx = s.retainContext(ctx)

	return s.setPosition(ctx, s.fullOpenPosition)
}

func (s *RelaysShutter) Close(ctx context.Context) error {
	logrus.Infof("%s: close", s.name)
	ctx = s.retainContext(ctx)

	return s.setPosition(ctx, s.fullClosePosition)
}

func (s *RelaysShutter) Stop(_ context.Context) error {
	logrus.Infof("%s: stop", s.name)

	if s.cancelCurrentContext != nil {
		s.cancelCurrentContext()
	}

	return nil
}
func (s *RelaysShutter) SetPosition(ctx context.Context, targetPosition int) error {
	logrus.Infof("%s: set targetPosition to %d", s.name, targetPosition)
	ctx = s.retainContext(ctx)

	if targetPosition > s.fullOpenPosition || targetPosition < s.fullClosePosition {
		return errors.Errorf(
			"%s: %d is out of range open/close targetPosition for (%d/%d)",
			s.name,
			targetPosition,
			s.fullOpenPosition,
			s.fullClosePosition,
		)
	}

	return s.setPosition(ctx, targetPosition)
}

func (s *RelaysShutter) setPosition(ctx context.Context, targetPosition int) error {
	logrus.Infof("%s: set targetPosition to %d", s.name, targetPosition)

	if targetPosition > s.fullOpenPosition || targetPosition < s.fullClosePosition {
		return errors.Errorf(
			"%s: %d is out of range open/close targetPosition for (%d/%d)",
			s.name,
			targetPosition,
			s.fullOpenPosition,
			s.fullClosePosition,
		)
	}

	go func() {
		if s.currentPosition == targetPosition {
			logrus.Debugf("%s: already on a position %d", s.name, targetPosition)
			return
		}

		// todo refactor

		diff := targetPosition - s.currentPosition
		if diff < 0 {
			diff = -diff
		}

		timeToMove := (s.timeToClose * time.Duration(diff)) / 100
		logrus.Debugf("%s: move by %d (%s)", s.name, diff, timeToMove.String())

		if targetPosition > s.currentPosition {
			s.currentState = shutterOpeningState
			s.updateHandler(s.currentState, s.currentPosition)
			if err := s.rUp.EnableFor(ctx, timeToMove); err != nil {
				logrus.Error(err)
			}
		} else {
			s.currentState = shutterClosingState
			s.updateHandler(s.currentState, s.currentPosition)
			if err := s.rDown.EnableFor(ctx, timeToMove); err != nil {
				logrus.Error(err)
			}
		}

		// todo update position ???

		if targetPosition == s.fullClosePosition {
			s.currentState = shutterClosedState
		} else {
			s.currentState = shutterOpenState
		}
		s.currentPosition = targetPosition
		s.updateHandler(s.currentState, s.currentPosition)

		logrus.Infof("%s: updated state %s, position %d", s.name, s.currentState, s.currentPosition)
	}()

	return nil
}
