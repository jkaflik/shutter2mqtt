package mqtt

import (
	"encoding/json"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
)

type haDevice struct {
	Identifiers  []string `json:"ids,omitempty"`
	Manufacturer string   `json:"mf,omitempty"`
	Model        string   `json:"mdl,omitempty"`
	Name         string   `json:"name,omitempty"`
	SWVersion    string   `json:"sw,omitempty"`
}

type haEntity struct {
	AvailabilityTopic string `json:"avty_t,omitempty"`
	UniqueID          string `json:"uniq_id,omitempty"`
	Name              string `json:"name,omitempty"`
	DeviceClass       string `json:"device_class,omitempty"`

	Device haDevice `json:"device,omitempty"`
}

type haCover struct {
	haEntity
	StateTopic       string `json:"stat_t"`
	CommandTopic     string `json:"cmd_t"`
	PositionTopic    string `json:"pos_t"`
	SetPositionTopic string `json:"set_pos_t"`
	PositionOpen     int    `json:"pos_open"`
	PositionClosed   int    `json:"pos_clsd"`
}

func NewHACoverFromMQTTBridge(bridge *Bridge) haCover {
	return haCover{
		haEntity: haEntity{
			//AvailabilityTopic: "",
			UniqueID:    bridge.shutter.Name(),
			Name:        bridge.shutter.Name(),
			DeviceClass: "shutter",

			Device: haDevice{
				Identifiers:  nil,
				Manufacturer: "Somfy",
				Model:        "Ilmo",
				Name:         bridge.shutter.Name(),
				SWVersion:    "shutters2mqtt",
			},
		},
		StateTopic:       bridge.StateTopic,
		CommandTopic:     bridge.CommandTopic,
		PositionTopic:    bridge.PositionTopic,
		SetPositionTopic: bridge.PositionChangeTopic,
		PositionOpen:     bridge.shutter.FullOpenPosition(),
		PositionClosed:   bridge.shutter.FullClosePosition(),
	}
}

func (e haEntity) Publish(client paho.Client, homeAssistantDiscoveryTopicPrefix string) error {
	return publishHAAutoDiscovery(client, homeAssistantDiscoveryTopicPrefix, e)
}

func publishHAAutoDiscovery(client paho.Client, homeAssistantDiscoveryTopicPrefix string, haEntity haEntity) error {
	topic := fmt.Sprintf("%s/cover/shutters2mqtt/%s/config", homeAssistantDiscoveryTopicPrefix, haEntity.Name)

	payload, err := json.Marshal(haEntity)
	if err != nil {
		return err
	}

	if token := client.Publish(topic, 0, true, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
