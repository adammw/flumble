package main

import (
	"flumble/pkg/flumble"
	"flumble/pkg/util"
	"fmt"
	"github.com/kolo/xmlrpc"
	"go.uber.org/zap"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
	"log"
	"net"

	_ "layeh.com/gumble/opus" // imported to enable opus audio codec
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	log := logger.Sugar()
	defer logger.Sync()

	config, err := flumble.BuildConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	flrigClient, err := xmlrpc.NewClient(fmt.Sprintf("http://%v", config.FlrigAddr), nil)
	if err != nil {
		log.Fatalf("can't initialize flrig client: %v", err)
	}

	var flrigVersion string
	err = flrigClient.Call("main.get_version", nil, &flrigVersion)
	if err != nil {
		log.Fatalf("can't get flrig version: %v", err)
	}
	log.Debugf("flrig version: %v", flrigVersion)

	keepAlive := make(chan bool)

	app := flumble.NewApp(config, logger, flrigClient)

	config.Gumble.Attach(gumbleutil.AutoBitrate)
	config.Gumble.Attach(gumbleutil.Listener{
		Disconnect: func(e *gumble.DisconnectEvent) {
			keepAlive <- true
		},
	})
	config.Gumble.AttachAudio(util.AudioListener{
		AudioStream: func(e *gumble.AudioStreamEvent) {
			// start a gofunc listening for audio packets
			go app.HandleAudioStream(e)
		},
	})

	log.Debugf("started")
	_, err = gumble.DialWithDialer(new(net.Dialer), config.MumbleAddr, config.Gumble, &config.MumbleTlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	<-keepAlive
}
