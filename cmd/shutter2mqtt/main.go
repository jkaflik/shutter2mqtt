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

	m := paho.NewClient(pahoOptsFromConfig())
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		logrus.Fatal(token.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	bridges := shutter2mqttFromConfig(ctx, m)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		oscall := <-c
		log.Printf("system call:%+v", oscall)
		cancel()
	}()

	for _, bridge := range bridges {
		if Cfg.HASS.Enabled {
			entity := mqtt.NewHACoverFromMQTTBridge(bridge)
			if err := mqtt.PublishHAAutoDiscovery(m, Cfg.HASS.TopicPrefix, entity); err != nil {
				logrus.Fatal(err)
			}

		}

		if err := bridge.Subscribe(ctx); err != nil {
			logrus.Error(err)
			cancel()
		}
	}

	<-ctx.Done()

	cleanupTime := time.Second
	logrus.Infof("cleanups for %s...", cleanupTime.String())
	time.Sleep(cleanupTime)
}
