package main

import (
	"context"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/jkaflik/shutter2mqtt/internal/mqtt"
	"github.com/jkaflik/shutter2mqtt/internal/shutter/driver/relay"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	opts := paho.NewClientOptions().
		SetClientID("shutters2mqtt").
		AddBroker("10.0.10.10:1883").
		SetUsername("panasonic").
		SetPassword("panasonic").
		SetConnectTimeout(time.Second).
		SetPingTimeout(time.Second).
		SetWriteTimeout(time.Second).
		SetAutoReconnect(true)

	m := paho.NewClient(opts)
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		logrus.Fatal(token.Error())
	}

	shutter := relay.NewRelaysShutter("shutter", &relay.Dumb{Name: "up"}, &relay.Dumb{Name: "down"}, 100, 0, time.Second*2)
	mqttBridge, err := mqtt.NewBridge(m, shutter)
	if err != nil {
		log.Fatal(err)
	}
	if err := mqttBridge.SetMetadata(map[string]string{
		"property": "value",
	}); err != nil {
		log.Fatal(err)
	}

	if err := mqtt.NewHACoverFromMQTTBridge(mqttBridge).Publish(m, "homeassistant"); err != nil {
		logrus.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		oscall := <-c
		log.Printf("system call:%+v", oscall)
		cancel()
	}()

	if err := mqttBridge.Subscribe(ctx); err != nil {
		logrus.Error(err)
		cancel()
	}

	for range ctx.Done() {
		return
	}
}
