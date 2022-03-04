package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jkaflik/shutter2mqtt/internal/shutter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strconv"
)

const (
	mqttOpenCmd  = "open"
	mqttCloseCmd = "close"
	mqttStopCmd  = "stop"
)

type Bridge struct {
	mqtt    mqtt.Client
	shutter shutter.Shutter

	StateTopic    string
	PositionTopic string
	MetadataTopic string

	CommandTopic        string
	PositionChangeTopic string
}

func NewBridge(mqtt mqtt.Client, shutter shutter.Shutter) *Bridge {
	bridge := &Bridge{mqtt: mqtt, shutter: shutter}
	bridge.StateTopic = fmt.Sprintf("shutters2mqtt/%s/state", shutter.Name())
	bridge.PositionTopic = fmt.Sprintf("shutters2mqtt/%s/position", shutter.Name())
	bridge.MetadataTopic = fmt.Sprintf("shutters2mqtt/%s/metadata", shutter.Name())
	bridge.CommandTopic = fmt.Sprintf("shutters2mqtt/%s/set", shutter.Name())
	bridge.PositionChangeTopic = fmt.Sprintf("shutters2mqtt/%s/position/set", shutter.Name())
	bridge.restorePosition()

	shutter.OnUpdate(bridge.onShutterUpdateHandler())

	return bridge
}

func (b *Bridge) SetMetadata(value interface{}) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if token := b.mqtt.Publish(b.MetadataTopic, 0, true, payload); token.Wait() && token.Error() != nil {
		return errors.Wrapf(token.Error(), "%s: MQTT metadata publish failed", b.shutter.Name())
	}

	return nil
}

func (b *Bridge) Subscribe(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				if token := b.mqtt.Unsubscribe(b.PositionChangeTopic, b.CommandTopic); token.Wait() && token.Error() != nil {
					logrus.Errorf("%s: MQTT topics unsubscribe failed: %s", b.shutter.Name(), token.Error())
				}
			}
		}
	}()

	if token := b.mqtt.Subscribe(b.CommandTopic, 0, b.onCommandHandler(ctx)); token.Wait() && token.Error() != nil {
		return errors.Wrapf(token.Error(), "%s: MQTT command topic subscription failed:", b.shutter.Name())
	}
	logrus.Infof("%s: MQTT command topic subscribed", b.shutter.Name())
	if token := b.mqtt.Subscribe(b.PositionChangeTopic, 0, b.onPositionChangeHandler(ctx)); token.Wait() && token.Error() != nil {
		return errors.Wrapf(token.Error(), "%s: MQTT position change topic subscription failed", b.shutter.Name())
	}
	logrus.Infof("%s: MQTT position change topic subscribed", b.shutter.Name())

	return nil
}

func (b *Bridge) onShutterUpdateHandler() shutter.ShutterUpdateHandler {
	return func(state string, position int) {
		if token := b.mqtt.Publish(b.StateTopic, 0, true, state); token.Wait() && token.Error() != nil {
			logrus.Errorf("%s: MQTT state publish failed: %s", b.shutter.Name(), token.Error())
		}
		if token := b.mqtt.Publish(b.PositionTopic, 0, true, fmt.Sprintf("%d", position)); token.Wait() && token.Error() != nil {
			logrus.Errorf("%s: MQTT position publish failed: %s", b.shutter.Name(), token.Error())
		}
	}
}

func (b *Bridge) onCommandHandler(ctx context.Context) mqtt.MessageHandler {
	return func(c mqtt.Client, msg mqtt.Message) {
		cmd := string(msg.Payload())
		switch cmd {
		case mqttOpenCmd:
			b.shutter.Open(ctx)
		case mqttCloseCmd:
			b.shutter.Close(ctx)
		case mqttStopCmd:
			b.shutter.Stop(ctx)
		default:
			logrus.Errorf("%s: MQTT unsupported %s command received", b.shutter.Name(), cmd)
		}
	}
}

func (b *Bridge) onPositionChangeHandler(ctx context.Context) mqtt.MessageHandler {
	return func(c mqtt.Client, msg mqtt.Message) {
		pos, err := strconv.Atoi(string(msg.Payload()))
		if err != nil {
			logrus.Error(err)
		}
		if err := b.shutter.SetPosition(ctx, pos); err != nil {
			logrus.Error(err)
		}
	}
}

func (b *Bridge) restorePosition() error {
	shutter, ok := b.shutter.(shutter.StatelessShutter)
	if !ok {
		logrus.Warnf("%s: MQTT position restore: shutter is not stateless", b.shutter.Name())
		return nil
	}

	restoreHandler := func(c mqtt.Client, msg mqtt.Message) {
		pos, err := strconv.Atoi(string(msg.Payload()))
		if err != nil {
			logrus.Error(err)
			return
		}
		if err := shutter.ResetPosition(pos); err != nil {
			logrus.Errorf("%s: MQTT position restore failed: %s", b.shutter.Name(), err)
			return
		}

		logrus.Infof("%s: MQTT position restored to %d", b.shutter.Name(), pos)

		if token := b.mqtt.Unsubscribe(b.PositionTopic); token.Wait() && token.Error() != nil {
			logrus.Errorf("%s: MQTT position restore topic unsubscribe failed: %s", b.shutter.Name(), token.Error())
			return
		}

		logrus.Debugf("%s: MQTT position restore topic unsubscribed", b.shutter.Name())
	}

	if token := b.mqtt.Subscribe(b.PositionTopic, 0, restoreHandler); token.Wait() && token.Error() != nil {
		return errors.Wrapf(token.Error(), "%s: MQTT position restore topic subscription failed:", b.shutter.Name())
	}

	return nil
}
