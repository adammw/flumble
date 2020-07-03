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
	"time"

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

	config.Gumble.Attach(gumbleutil.AutoBitrate)
	config.Gumble.Attach(gumbleutil.Listener{
		Disconnect: func(e *gumble.DisconnectEvent) {
			keepAlive <- true
		},
	})
	config.Gumble.AttachAudio(util.AudioListener{
		AudioStream: func(e *gumble.AudioStreamEvent) {
			log.Debugw("AudioStream", "user", e.User.Name, "audiointerval", e.Client.Config.AudioInterval)

			// start a gofunc listening for audio packets
			go func() {
				talking := false
				closed := false
				lastPkt := time.Now()
				for {
					select {
					case audioPkt, more := <-e.C:
						if !talking {
							log.Debugw("start talking", "user", e.User.Name, "len", len(audioPkt.AudioBuffer))
							if e.User.Name != config.IgnoreUsername {
								err = flrigClient.Call("rig.set_ptt", 1, nil)
								if err != nil {
									log.Error(err)
								}
							}
						}
						talking = true
						closed = !more // handle closed audio channel
						lastPkt = time.Now()

					case <-time.After(config.AudioTimeout):
						if talking {

							log.Debugw("stop talking", "user", e.User.Name, "timeSinceLast", time.Now().Sub(lastPkt).Seconds())
							if e.User.Name != config.IgnoreUsername {
								err = flrigClient.Call("rig.set_ptt", 0, nil)
								if err != nil {
									log.Error(err)
								}
							}
						}
						talking = false

						if (closed) {
							break;
						}
					}
				}
			}()
		},
	})

	log.Debugf("started")
	_, err = gumble.DialWithDialer(new(net.Dialer), config.MumbleAddr, config.Gumble, &config.MumbleTlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	<-keepAlive
}
