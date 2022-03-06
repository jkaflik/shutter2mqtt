package relay

import (
	"context"
	"time"

	"github.com/jkaflik/shutter2mqtt/internal/shutter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RelaysShutter struct {
	rUp   Relay
	rDown Relay

	name              string
	fullOpenPosition  int
	fullClosePosition int
	timeToClose       time.Duration

	updateHandler shutter.ShutterUpdateHandler

	currentState    string
	currentPosition int

	cancelCurrentContext context.CancelFunc
}

func (s *RelaysShutter) ResetPosition(position int) error {
	s.currentPosition = position
	if s.currentPosition == s.fullClosePosition {
		s.currentState = shutter.ShutterClosedState
	}
	s.currentState = shutter.ShutterOpenState

	return nil
}

func NewRelaysShutter(name string, up Relay, down Relay, fullOpenPosition int, fullClosePosition int, timeToClose time.Duration) *RelaysShutter {
	s := &RelaysShutter{rUp: up, rDown: down, name: name, fullOpenPosition: fullOpenPosition, fullClosePosition: fullClosePosition, timeToClose: timeToClose}
	s.currentState = shutter.ShutterOpenState
	s.currentPosition = s.fullClosePosition
	return s
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

func (s *RelaysShutter) OnUpdate(h shutter.ShutterUpdateHandler) {
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

	if s.currentPosition == s.fullClosePosition {
		s.currentState = shutter.ShutterClosedState
	} else {
		s.currentState = shutter.ShutterOpenState
	}

	s.updateHandler(s.currentState, s.currentPosition)

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

		// todo refactor
		var relay Relay
		if targetPosition > s.currentPosition {
			s.currentState = shutter.ShutterOpeningState
			relay = s.rUp
		} else {
			s.currentState = shutter.ShutterClosingState
			relay = s.rDown
		}

		go s.calculatePositionDuringMove(ctx, targetPosition, timeToMove)

		logrus.Debugf("%s: enable relay for %s", s.name, timeToMove.String())
		s.updateHandler(s.currentState, s.currentPosition)
		if err := relay.EnableFor(ctx, timeToMove); err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				logrus.Infof("%s: set position %d canceled", s.name, targetPosition)
			} else {
				logrus.Errorf("%s: enable relay error: %s", s.name, err)
			}
			return
		}

		if targetPosition == s.fullClosePosition {
			s.currentState = shutter.ShutterClosedState
		} else {
			s.currentState = shutter.ShutterOpenState
		}
		s.currentPosition = targetPosition
		s.updateHandler(s.currentState, s.currentPosition)

		logrus.Infof("%s: updated state %s, position %d", s.name, s.currentState, s.currentPosition)
	}()

	return nil
}

func (s *RelaysShutter) calculatePositionDuringMove(ctx context.Context, targetPosition int, timeToMove time.Duration) {
	logrus.Debugf("%s: begin position calculation", s.name)
	s.updateHandler(s.currentState, s.currentPosition)

	after := time.After(timeToMove)
	every := time.NewTicker(s.timeToClose / time.Duration(s.fullOpenPosition-s.fullClosePosition))
	defer every.Stop()
	for {
		select {
		case <-after:
			logrus.Debugf("%s: timeout position calculation", s.name)
			return
		case <-ctx.Done():
			logrus.Debugf("%s: exit position calculation", s.name)
			return
		case <-every.C:
			if s.currentPosition < targetPosition {
				logrus.Tracef("%s: increase position", s.name)
				s.currentPosition++
			} else {
				logrus.Tracef("%s: decrease position", s.name)
				s.currentPosition--
			}

			s.updateHandler(s.currentState, s.currentPosition)
		}
	}
}
