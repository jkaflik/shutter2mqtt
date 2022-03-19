package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/jkaflik/shutter2mqtt/internal/mqtt"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	configPath := flag.String("config", "config.yaml", "config.yaml file path")

	if err := configLoader.Load(); err != nil {
		logrus.Fatal(err)
	}
	loadConfigFromYamlFile(*configPath)

	level, err := logrus.ParseLevel(Cfg.LogLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(level)

	ctx, cancel := context.WithCancel(context.Background())
	var bridges []*mqtt.Bridge
	cfg := pahoOptsFromConfig()
	cfg.OnConnect = func(m paho.Client) {
		logrus.Info("MQTT broker connected")
		subscribe(ctx, m, bridges)
	}
	cfg.OnConnectionLost = func(_ paho.Client, err error) {
		logrus.Errorf("MQTT broker connection lost: %s", err.Error())
	}

	m := paho.NewClient(cfg)
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		logrus.Fatal(token.Error())
	}

	bridges = shutter2mqttFromConfig(ctx, m)
	subscribe(ctx, m, bridges)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		oscall := <-c
		log.Printf("system call:%+v", oscall)
		cancel()
	}()

	<-ctx.Done()

	cleanupTime := time.Second
	logrus.Infof("cleanups for %s...", cleanupTime.String())
	time.Sleep(cleanupTime)
}

func subscribe(ctx context.Context, m paho.Client, bridges []*mqtt.Bridge) {
	for _, bridge := range bridges {
		if Cfg.HASS.Enabled {
			entity := mqtt.NewHACoverFromMQTTBridge(bridge)
			if err := mqtt.PublishHAAutoDiscovery(m, Cfg.HASS.TopicPrefix, entity); err != nil {
				logrus.Fatal(err)
			}

		}

		if err := bridge.Subscribe(ctx); err != nil {
			logrus.Error(err)
		}
	}
}
