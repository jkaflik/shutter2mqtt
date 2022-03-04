package main

import (
	"context"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/jkaflik/shutter2mqtt/internal/mqtt"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"os/signal"
)

func main() {
	if err := configLoader.Load(); err != nil {
		logrus.Fatal(err)
	}

	level, err := logrus.ParseLevel(Cfg.logLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(level)

	m := paho.NewClient(pahoOptsFromConfig())
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		logrus.Fatal(token.Error())
	}

	bridges := shutter2mqttFromConfig(m)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		oscall := <-c
		log.Printf("system call:%+v", oscall)
		cancel()
	}()

	for _, bridge := range bridges {
		if Cfg.homeAssistant.enabled {
			if err := mqtt.NewHACoverFromMQTTBridge(bridge).Publish(m, Cfg.homeAssistant.topicPrefix); err != nil {
				logrus.Fatal(err)
			}

		}

		if err := bridge.Subscribe(ctx); err != nil {
			logrus.Error(err)
			cancel()
		}
	}

	for range ctx.Done() {
		return
	}
}
