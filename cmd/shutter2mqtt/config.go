package main

import (
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
		Mcp23017 map[int]struct {
			Bus     uint8 `yaml:"bus"`
			Address uint8 `yaml:"address"`

			d *mcp23017.Device `yaml:"-"`
		} `yaml:"mcp23017"`
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

func shutter2mqttFromConfig(client paho.Client) (bridges []*mqtt.Bridge) {
	for _, cfg := range Cfg.Shutters {
		s := shutterFromConfig(cfg)
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

func shutterFromConfig(cfg cfgShutter) shutter.Shutter {
	if cfg.Kind == "relays" {
		return relay.NewRelaysShutter(
			cfg.Name,
			relayFromConfig(cfg.Driver.Relays.Up),
			relayFromConfig(cfg.Driver.Relays.Down),
			cfg.Driver.Relays.FullOpenPosition,
			cfg.Driver.Relays.FullClosePosition,
			cfg.Driver.Relays.TimeToClose,
		)
	}

	logrus.Fatalf("%s is not supported shutter kind", cfg.Kind)
	return nil
}

func relayFromConfig(cfg cfgRelay) relay.Relay {
	if cfg.Kind == "wired" {
		return &relay.Wired{
			Pin:          wiredRelaySetPinFromConfig(cfg.Pin),
			NormalClosed: cfg.NormalClosed,
		}
	}

	if cfg.Kind == "dumb" {
		return &relay.Dumb{Name: cfg.Kind}
	}

	logrus.Fatalf("%s is not supported relay kind", cfg.Kind)
	return nil
}

func wiredRelaySetPinFromConfig(cfg cfgWiredRelaySetPin) relay.SetPin {
	if cfg.Kind == "mcp23017" {
		device := mcp23017DeviceFromConfigByID(cfg.Mcp23017)

		p, err := relay.NewMcp23017Pin(device, cfg.Pin)
		if err != nil {
			logrus.Fatal(err)
		}
		return p
	}

	logrus.Fatalf("%s is not supported wired relay set pin kind", cfg.Kind)
	return nil
}

func mcp23017DeviceFromConfigByID(id int) *mcp23017.Device {
	if Cfg.Drivers.Relay.Mcp23017 == nil {
		logrus.Fatal("drivers.relay.mcp23017 not defined")
	}

	cfg, found := Cfg.Drivers.Relay.Mcp23017[id]
	if !found {
		logrus.Fatalf("%d is not valid defined drivers.relay.mcp23017", id)
		return nil
	}

	if cfg.d == nil {
		var err error
		cfg.d, err = mcp23017.Open(cfg.Bus, cfg.Address)
		if err != nil {
			logrus.Fatal(err)
		}
	}

	return cfg.d
}
