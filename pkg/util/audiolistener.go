package util

import "layeh.com/gumble/gumble"

// Listener is a struct that implements the gumble.EventListener interface. The
// corresponding event function in the struct is called if it is non-nil.
type AudioListener struct {
	AudioStream			func(e *gumble.AudioStreamEvent)
}

func (a AudioListener) OnAudioStream(e *gumble.AudioStreamEvent) {
	if a.AudioStream != nil {
		a.AudioStream(e)
	}
}

var _ gumble.AudioListener = (*AudioListener)(nil)

