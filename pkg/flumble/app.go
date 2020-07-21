package flumble

import (
	"github.com/kolo/xmlrpc"
	"go.uber.org/zap"
	"layeh.com/gumble/gumble"
	"time"
)

type App struct {
	config 		*Config
	log 		*zap.SugaredLogger
	flrigClient *xmlrpc.Client
}

func NewApp(config *Config, logger *zap.Logger, flrigClient *xmlrpc.Client) *App {
	return &App{
		config: config,
		log: logger.Sugar(),
		flrigClient: flrigClient,
	}
}

func (a *App) HandleAudioStream(e *gumble.AudioStreamEvent) {
	a.log.Debugw("AudioStream", "user", e.User.Name, "audiointerval", e.Client.Config.AudioInterval)

	talking := false
	closed := false
	inhibit := false

	startTalkingTime := time.Now()
	lastPkt := time.Now()

	for {
		select {
		case audioPkt, more := <-e.C:
			// new audio packet -> assume started talking
			if !talking {
				if inhibit {
					a.log.Debugw("audio detected but tx inhibited", "talkTime", time.Now().Sub(startTalkingTime))
				} else {
					a.log.Debugw("started talking", "user", e.User.Name, "len", len(audioPkt.AudioBuffer))
					startTalkingTime = time.Now()
					if e.User.Name != a.config.IgnoreUsername {
						err := a.flrigClient.Call("rig.set_ptt", 1, nil)
						if err != nil {
							a.log.Error(err)
						}
					}
				}
			}

			talking = true
			closed = !more // handle closed audio channel
			lastPkt = time.Now()

			// check for exceeding maximum transmission time
			if time.Now().Sub(startTalkingTime) > a.config.MaxTransmitTime {
				a.log.Debugw("maximum transmission time exceeded", "talkTime", time.Now().Sub(startTalkingTime))
				inhibit = true

				err := a.flrigClient.Call("rig.set_ptt", 0, nil)
				if err != nil {
					a.log.Error(err)
				}
			}

		case <-time.After(a.config.AudioTimeout):
			// no audio packet -> assume stopped talking
			if talking {
				a.log.Debugw("stopped talking", "user", e.User.Name, "timeSinceLast", time.Now().Sub(lastPkt))
				if e.User.Name != a.config.IgnoreUsername {
					err := a.flrigClient.Call("rig.set_ptt", 0, nil)
					if err != nil {
						a.log.Error(err)
					}
				}
			}
			talking = false

			// check if silence is sufficient to turn off inhibit
			if inhibit && time.Now().Sub(lastPkt) > a.config.MinSilenceTime {
				a.log.Debugw("minimum silence time met, may resume transmission", "silence time", time.Now().Sub(lastPkt))
				inhibit = false
			}

			if (closed) {
				break;
			}
		}
	}
}
