package mqtt

import (
	"encoding/json"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
)

type HACover struct {
	AvailabilityTopic string `json:"avty_t,omitempty"`
	StateTopic        string `json:"stat_t"`
	CommandTopic      string `json:"cmd_t"`
	PositionTopic     string `json:"pos_t"`
	SetPositionTopic  string `json:"set_pos_t"`
	PositionOpen      int    `json:"pos_open"`
	PositionClosed    int    `json:"pos_clsd"`

	UniqueID    string `json:"uniq_id,omitempty"`
	Name        string `json:"name,omitempty"`
	DeviceClass string `json:"device_class,omitempty"`
	Device      struct {
		Identifiers  []string `json:"ids,omitempty"`
		Manufacturer string   `json:"mf,omitempty"`
		Model        string   `json:"mdl,omitempty"`
		Name         string   `json:"name,omitempty"`
		SWVersion    string   `json:"sw,omitempty"`
	} `json:"device,omitempty"`
}

func newHACoverFromMQTTBridge(bridge *Bridge) HACover {
	return HACover{
		//AvailabilityTopic: "", // todo
		StateTopic:       bridge.StateTopic,
		CommandTopic:     bridge.CommandTopic,
		PositionTopic:    bridge.PositionTopic,
		SetPositionTopic: bridge.PositionChangeTopic,
		PositionOpen:     bridge.shutter.FullOpenPosition(),
		PositionClosed:   bridge.shutter.FullClosePosition(),
		UniqueID:         bridge.shutter.Name(),
		Name:             bridge.shutter.Name(),
		DeviceClass:      "shutter",
		Device: struct {
			Identifiers  []string `json:"ids,omitempty"`
			Manufacturer string   `json:"mf,omitempty"`
			Model        string   `json:"mdl,omitempty"`
			Name         string   `json:"name,omitempty"`
			SWVersion    string   `json:"sw,omitempty"`
		}{
			Identifiers:  nil,
			Manufacturer: "Somfy",
			Model:        "Ilmo",
			Name:         bridge.shutter.Name(),
			SWVersion:    "shutters2mqtt",
		},
	}
}

func publishHAAutoDiscovery(client paho.Client, homeAssistantDiscoveryTopicPrefix string, haCover HACover) error {
	topic := fmt.Sprintf("%s/cover/shutters2mqtt/%s/config", homeAssistantDiscoveryTopicPrefix, haCover.Name)

	payload, err := json.Marshal(haCover)
	if err != nil {
		return err
	}

	if token := client.Publish(topic, 0, true, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
