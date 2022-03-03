package main

import (
	"context"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	opts := mqtt.NewClientOptions().
		SetClientID("shutters2mqtt").
		AddBroker("10.0.10.10:1883").
		SetUsername("panasonic").
		SetPassword("panasonic").
		SetConnectTimeout(time.Second).
		SetPingTimeout(time.Second).
		SetWriteTimeout(time.Second).
		SetAutoReconnect(true)

	m := mqtt.NewClient(opts)
	if token := m.Connect(); token.Wait() && token.Error() != nil {
		logrus.Fatal(token.Error())
	}

	salonL := NewRelaysShutter("salon_l", &DumbRelay{"up"}, &DumbRelay{"down"}, 100, 0, time.Second*2)
	salonLBridge := NewMQTTBridge(m, salonL)
	salonLBridge.SetMetadata(map[string]string{
		"azimuth":      "253",
		"windowTop":    "240",
		"windowBottom": "0",
	})

	salonP := NewRelaysShutter("salon_p", &DumbRelay{"up"}, &DumbRelay{"down"}, 100, 0, time.Second*2)
	salonPBridge := NewMQTTBridge(m, salonP)
	salonPBridge.SetMetadata(map[string]string{
		"azimuth":      "253",
		"windowTop":    "240",
		"windowBottom": "0",
	})

	kuchniaL := NewRelaysShutter("kuchnia_l", &DumbRelay{"up"}, &DumbRelay{"down"}, 100, 0, time.Second*2)
	kuchniaLBridge := NewMQTTBridge(m, kuchniaL)
	kuchniaLBridge.SetMetadata(map[string]string{
		"azimuth":      "76",
		"windowTop":    "240",
		"windowBottom": "130",
	})

	kuchniaP := NewRelaysShutter("kuchnia_p", &DumbRelay{"up"}, &DumbRelay{"down"}, 100, 0, time.Second*2)
	kuchniaPBridge := NewMQTTBridge(m, kuchniaP)
	kuchniaPBridge.SetMetadata(map[string]string{
		"azimuth":      "162",
		"windowTop":    "240",
		"windowBottom": "130",
	})

	// 76 kuchnia l
	// 162 kuchnia p

	//if err := publishHAAutoDiscovery(m, "homeassistant", newHACoverFromMQTTBridge(bridge)); err != nil {
	//	logrus.Fatal(err)
	//}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		oscall := <-c
		log.Printf("system call:%+v", oscall)
		cancel()
	}()

	if err := salonPBridge.Subscribe(ctx); err != nil {
		logrus.Error(err)
		cancel()
	}

	if err := salonLBridge.Subscribe(ctx); err != nil {
		logrus.Error(err)
		cancel()
	}

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
