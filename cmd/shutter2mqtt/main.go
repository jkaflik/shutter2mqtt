package main

import (
	"context"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/jkaflik/shutter2mqtt/internal/mqtt"
	relay2 "github.com/jkaflik/shutter2mqtt/internal/shutter/driver/relay"
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

	salonL := relay2.NewRelaysShutter("salon_l", &relay2.Dumb{"up"}, &relay2.Dumb{"down"}, 100, 0, time.Second*2)
	salonLBridge := mqtt.NewBridge(m, salonL)
	salonLBridge.SetMetadata(map[string]string{
		"azimuth":      "253",
		"windowTop":    "240",
		"windowBottom": "0",
	})

	salonP := relay2.NewRelaysShutter("salon_p", &relay2.Dumb{"up"}, &relay2.Dumb{"down"}, 100, 0, time.Second*2)
	salonPBridge := mqtt.NewBridge(m, salonP)
	salonPBridge.SetMetadata(map[string]string{
		"azimuth":      "253",
		"windowTop":    "240",
		"windowBottom": "0",
	})

	kuchniaL := relay2.NewRelaysShutter("kuchnia_l", &relay2.Dumb{"up"}, &relay2.Dumb{"down"}, 100, 0, time.Second*2)
	kuchniaLBridge := mqtt.NewBridge(m, kuchniaL)
	kuchniaLBridge.SetMetadata(map[string]string{
		"azimuth":      "76",
		"windowTop":    "240",
		"windowBottom": "130",
	})

	kuchniaP := relay2.NewRelaysShutter("kuchnia_p", &relay2.Dumb{"up"}, &relay2.Dumb{"down"}, 100, 0, time.Second*2)
	kuchniaPBridge := mqtt.NewBridge(m, kuchniaP)
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
