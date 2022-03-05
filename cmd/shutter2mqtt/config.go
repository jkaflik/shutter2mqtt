package main

import (
	"context"
	"os"
	"time"

	"github.com/cristalhq/aconfig"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/jkaflik/shutter2mqtt/internal/mqtt"
	"github.com/jkaflik/shutter2mqtt/internal/shutter"
	"github.com/jkaflik/shutter2mqtt/internal/shutter/driver/relay"
	"github.com/racerxdl/go-mcp23017"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type cfgWiredRelaySetPin struct {
	Kind string `yaml:"kind"`

	Pin uint8 `yaml:"pin"`

	Mcp23017 int `yaml:"mcp23017"`
}

type cfgRelay struct {
	Kind string `yaml:"kind"`

	Pin          cfgWiredRelaySetPin `yaml:"pin"`
	NormalClosed bool                `yaml:"normal_closed"`
}

type cfgShutterMQTTBridge struct {
	Metadata map[string]interface{} `yaml:"metadata"`
}

type cfgShutterDriverRelays struct {
	Up   cfgRelay `yaml:"up"`
	Down cfgRelay `yaml:"down"`

	FullOpenPosition  int           `yaml:"full_open_position" default:"100"`
	FullClosePosition int           `yaml:"full_close_position" default:"0"`
	TimeToClose       time.Duration `yaml:"time_to_close" default:"1m"`
}

type cfgShutterDriver struct {
	Relays cfgShutterDriverRelays `yaml:"relays"`
}

type cfgShutter struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`

	MQTTBridge cfgShutterMQTTBridge `yaml:"mqtt_bridge"`

	Driver cfgShutterDriver `yaml:"driver"`
}

type cfgDrivers struct {
	Relay struct {
		Pool     int `yaml:"pool" default:"0"`
		Mcp23017 map[int]struct {
			Bus          uint8 `yaml:"bus" default:"1"`
			DeviceNumber uint8 `yaml:"device_number" default:"0"`
		} `yaml:""`
	} `yaml:"relay"`
}

type cfgMQTT struct { // todo more fields
	ClientID string `yaml:"client_id" default:"shutter2mqtt" env:"CLIENT_ID"`
	Broker   string `yaml:"broker" default:"127.0.0.1:1883" env:"BROKER"`
	Username string `yaml:"username" env:"USERNAME"`
	Password string `yaml:"password" env:"PASSWORD"`
}

type cfgHASS struct {
	Enabled     bool   `yaml:"enabled" default:"true" env:"ENABLED"`
	TopicPrefix string `yaml:"topic_prefix" default:"homeassistant" env:"TOPIC_PREFIX"`
}

var Cfg struct {
	LogLevel string `yaml:"log_level" default:"info" env:"LOG_LEVEL"`

	MQTT cfgMQTT `yaml:"mqtt" env:"MQTT"`
	HASS cfgHASS `yaml:"hass" env:"HASS"`

	Shutters []cfgShutter `yaml:"shutters"`

	Drivers cfgDrivers `yaml:"drivers"`
}

var configLoader = aconfig.LoaderFor(&Cfg, aconfig.Config{
	EnvPrefix: "S2M",
})

var relaysPool chan struct{}

func loadConfigFromYamlFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&Cfg); err != nil {
		logrus.Fatal(err)
		return
	}

	if Cfg.Drivers.Relay.Pool > 0 {
		relaysPool = make(chan struct{}, Cfg.Drivers.Relay.Pool)
	}
}

func pahoOptsFromConfig() *paho.ClientOptions {
	return paho.NewClientOptions().
		SetClientID(Cfg.MQTT.ClientID).
		AddBroker(Cfg.MQTT.Broker).
		SetUsername(Cfg.MQTT.Username).
		SetPassword(Cfg.MQTT.Password).
		SetConnectTimeout(time.Second).
		SetPingTimeout(time.Second).
		SetWriteTimeout(time.Second).
		SetAutoReconnect(true)
}

func shutter2mqttFromConfig(ctx context.Context, client paho.Client) (bridges []*mqtt.Bridge) {
	for _, cfg := range Cfg.Shutters {
		s := shutterFromConfig(ctx, cfg)
		bridge, err := mqtt.NewBridge(client, s)
		if err != nil {
			logrus.Fatal(err)
			continue
		}
		if err := bridge.SetMetadata(cfg.MQTTBridge.Metadata); err != nil {
			logrus.Fatal(err)
			continue
		}
		bridges = append(bridges, bridge)
	}

	return bridges
}

func shutterFromConfig(ctx context.Context, cfg cfgShutter) shutter.Shutter {
	if cfg.Kind == "relays" {
		return relay.NewRelaysShutter(
			cfg.Name,
			relayFromConfig(ctx, cfg.Driver.Relays.Up),
			relayFromConfig(ctx, cfg.Driver.Relays.Down),
			cfg.Driver.Relays.FullOpenPosition,
			cfg.Driver.Relays.FullClosePosition,
			cfg.Driver.Relays.TimeToClose,
		)
	}

	logrus.Fatalf("%s is not supported shutter kind", cfg.Kind)
	return nil
}

func relayFromConfig(ctx context.Context, cfg cfgRelay) relay.Relay {
	if cfg.Kind == "wired" {
		return wrapRelayWithPoolProxy(&relay.Wired{
			Pin:          wiredRelaySetPinFromConfig(ctx, cfg.Pin),
			NormalClosed: cfg.NormalClosed,
		})
	}

	if cfg.Kind == "dumb" {
		return wrapRelayWithPoolProxy(&relay.Dumb{Name: cfg.Kind})
	}

	logrus.Fatalf("%s is not supported relay kind", cfg.Kind)
	return nil
}

func wrapRelayWithPoolProxy(r relay.Relay) relay.Relay {
	if relaysPool == nil {
		return r
	}

	return relay.NewPoolProxy(r, relaysPool)
}

func wiredRelaySetPinFromConfig(ctx context.Context, cfg cfgWiredRelaySetPin) relay.SetPin {
	if cfg.Kind == "mcp23017" {
		device := mcp23017DeviceFromConfigByID(ctx, cfg.Mcp23017)

		p, err := relay.NewMcp23017Pin(device, cfg.Pin)
		if err != nil {
			logrus.Fatal(err)
		}
		return p
	}

	logrus.Fatalf("%s is not supported wired relay set pin kind", cfg.Kind)
	return nil
}

var mcpDevices = map[int]*mcp23017.Device{}

func mcp23017DeviceFromConfigByID(ctx context.Context, id int) *mcp23017.Device {
	if Cfg.Drivers.Relay.Mcp23017 == nil {
		logrus.Fatal("drivers.relay.mcp23017 not defined")
	}

	cfg, found := Cfg.Drivers.Relay.Mcp23017[id]
	if !found {
		logrus.Fatalf("%d is not valid defined drivers.relay.mcp23017", id)
		return nil
	}

	dev := mcpDevices[id]
	if dev == nil {
		var err error
		dev, err = mcp23017.Open(cfg.Bus, cfg.DeviceNumber)
		if err != nil {
			logrus.Fatal(err)
		}
		go func() {
			<-ctx.Done()
			if err := dev.Close(); err != nil {
				logrus.Errorf("mcp23017: close failed %s", err)
				return
			}

			logrus.Infof("mcp23017: close")
		}()
		if err := dev.Reset(); err != nil {
			logrus.Fatal(err)
		}

		mcpDevices[id] = dev
	}

	return dev
}
