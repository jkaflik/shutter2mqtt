package main

import (
	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigyaml"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/jkaflik/shutter2mqtt/internal/mqtt"
	"github.com/jkaflik/shutter2mqtt/internal/shutter"
	"github.com/jkaflik/shutter2mqtt/internal/shutter/driver/relay"
	"github.com/racerxdl/go-mcp23017"
	"github.com/sirupsen/logrus"
	"time"
)

type cfgWiredRelaySetPin struct {
	kind string

	pin uint8

	mcp23017 int
}

type cfgRelay struct {
	kind string

	pin          cfgWiredRelaySetPin
	normalClosed bool
}

type cfgShutter struct {
	name string
	kind string

	mqttBridge struct {
		metadata map[string]interface{}
	}

	driver struct {
		relays struct {
			up   cfgRelay
			down cfgRelay

			fullOpenPosition  int
			fullClosePosition int
			timeToClose       time.Duration
		}
	}
}

type cfgDriver struct {
	relay struct {
		mcp23017 map[int]struct {
			bus  uint8
			addr uint8

			d *mcp23017.Device
		}
	}
}

var Cfg struct {
	logLevel string

	mqtt struct { // todo more fields
		clientID string
		broker   string
		username string
		password string
	}

	homeAssistant struct {
		enabled     bool
		topicPrefix string
	}

	shutters []cfgShutter

	driver cfgDriver
}

var configLoader = aconfig.LoaderFor(&Cfg, aconfig.Config{
	EnvPrefix:  "S2M",
	FlagPrefix: "app",
	Files:      []string{"config.yaml"},
	FileDecoders: map[string]aconfig.FileDecoder{
		".yaml": aconfigyaml.New(),
	},
})

func pahoOptsFromConfig() *paho.ClientOptions {
	return paho.NewClientOptions().
		SetClientID(Cfg.mqtt.clientID).
		AddBroker(Cfg.mqtt.broker).
		SetUsername(Cfg.mqtt.username).
		SetPassword(Cfg.mqtt.password).
		SetConnectTimeout(time.Second).
		SetPingTimeout(time.Second).
		SetWriteTimeout(time.Second).
		SetAutoReconnect(true)
}

func shutter2mqttFromConfig(client paho.Client) (bridges []*mqtt.Bridge) {
	for _, cfg := range Cfg.shutters {
		s := shutterFromConfig(cfg)
		bridge, err := mqtt.NewBridge(client, s)
		if err != nil {
			logrus.Fatal(err)
			continue
		}
		if err := bridge.SetMetadata(cfg.mqttBridge.metadata); err != nil {
			logrus.Fatal(err)
			continue
		}
		bridges = append(bridges, bridge)
	}

	return bridges
}

func shutterFromConfig(cfg cfgShutter) shutter.Shutter {
	if cfg.kind == "relays" {
		return relay.NewRelaysShutter(
			cfg.name,
			relayFromConfig(cfg.driver.relays.up),
			relayFromConfig(cfg.driver.relays.down),
			cfg.driver.relays.fullOpenPosition,
			cfg.driver.relays.fullClosePosition,
			cfg.driver.relays.timeToClose,
		)
	}

	logrus.Fatalf("%s is not supported shutter kind", cfg.kind)
	return nil
}

func relayFromConfig(cfg cfgRelay) relay.Relay {
	if cfg.kind == "wired" {
		return &relay.Wired{
			Pin:          wiredRelaySetPinFromConfig(cfg.pin),
			NormalClosed: cfg.normalClosed,
		}
	}

	logrus.Fatalf("%s is not supported relay kind", cfg.kind)
	return nil
}

func wiredRelaySetPinFromConfig(cfg cfgWiredRelaySetPin) relay.SetPin {
	if cfg.kind == "mcp23017" {
		device := mcp23017DeviceFromConfigByID(cfg.mcp23017)

		p, err := relay.NewMcp23017Pin(device, cfg.pin)
		if err != nil {
			logrus.Fatal(err)
		}
		return p
	}

	logrus.Fatalf("%s is not supported wired relay set pin kind", cfg.kind)
	return nil
}

func mcp23017DeviceFromConfigByID(id int) *mcp23017.Device {
	cfg, found := Cfg.driver.relay.mcp23017[id]
	if !found {
		logrus.Fatalf("%d is not valid defined mcp23017.Device", id)
		return nil
	}

	if cfg.d == nil {
		var err error
		cfg.d, err = mcp23017.Open(cfg.bus, cfg.addr)
		if err != nil {
			logrus.Fatal(err)
		}
	}

	return cfg.d
}
