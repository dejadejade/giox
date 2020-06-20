package main

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dejadejade/giox/fn"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

// AudioPlayerResource holds the widget resources
type AudioPlayerResource struct {
	PlayIcon  *widget.Icon
	StopIcon  *widget.Icon
	PauseIcon *widget.Icon
}

type command uint8

const (
	cmdStop command = iota
	cmdPause
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// AudioPlayer holds the state of the player
type AudioPlayer struct {
	playBtn widget.Clickable
	stopBtn widget.Clickable

	res *AudioPlayerResource

	notify func()
	file   string
	data   []byte
	src    io.ReadCloser

	started bool
	playing bool

	duration time.Duration
	progress int
	position string

	format   beep.Format
	streamer beep.StreamSeekCloser
	done     chan command
	ctrl     *beep.Ctrl
}

// NewAudioPlayer returns the player
func NewAudioPlayer(file string, res *AudioPlayerResource, notify func()) *AudioPlayer {
	p := &AudioPlayer{file: file, res: res, done: make(chan command), notify: notify}
	p.position = "00:00:00"
	return p
}

func NewAudioPlayerWithData(data []byte, res *AudioPlayerResource, notify func()) *AudioPlayer {
	p := &AudioPlayer{data: data, res: res, done: make(chan command), notify: notify}
	p.position = "00:00:00"
	return p
}

func format(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (p *AudioPlayer) updateProgress(pos int) {
	if !p.playing {
		return
	}
	x := p.format.SampleRate.D(pos).Round(time.Second)
	p.position = format(p.duration - x)
	p.progress = int(x.Seconds()) * 100 / int(p.duration.Seconds())
	p.notify()
}

func (p *AudioPlayer) initialize() error {
	var err error
	if p.file != "" {
		if p.src, err = os.Open(p.file); err != nil {
			return err
		}
	} else if p.data != nil {
		p.src = ioutil.NopCloser(bytes.NewReader(p.data))
	}

	p.streamer, p.format, err = mp3.Decode(p.src)
	if err != nil {
		return err
	}

	p.ctrl = &beep.Ctrl{Streamer: beep.Seq(p.streamer, beep.Callback(func() {
		p.done <- cmdStop
	})), Paused: false}

	if err = speaker.Init(p.format.SampleRate, p.format.SampleRate.N(time.Second/10)); err != nil {
		return err
	}

	p.duration = p.format.SampleRate.D(p.streamer.Len()).Round(time.Second)
	p.position = format(p.duration)
	return nil
}

func (p *AudioPlayer) stop() {
	if p.started {
		p.done <- cmdStop
	}
}

func (p *AudioPlayer) updateThread() {
	for {
		select {
		case cmd := <-p.done:
			if cmd == cmdStop {
				speaker.Close()
				p.started = false
				p.progress = 0
				p.position = "00:00:00"
			}
			p.playing = false
			p.notify()
			return
		case <-time.After(time.Second):
			speaker.Lock()
			pos := p.streamer.Position()
			speaker.Unlock()
			p.updateProgress(pos)
		}
	}
}

func (p *AudioPlayer) start() {
	if p.started {
		return
	}

	if err := p.initialize(); err != nil {
		log.Printf("Failed to start: %v\n", err)
	}

	speaker.Play(p.ctrl)
	p.started = true
	p.playing = true
	go func() { p.updateThread() }()
}

func (p *AudioPlayer) playPause() {
	speaker.Lock()
	p.ctrl.Paused = !p.ctrl.Paused
	speaker.Unlock()

	p.playing = !p.playing
	if p.playing {
		go func() { p.updateThread() }()
	} else {
		p.done <- cmdPause
	}
}

//Layout draws the player controls
func (p *AudioPlayer) Layout(gtx C) D {
	if p.playBtn.Clicked() {
		go func() {
			if !p.started {
				p.start()
			} else {
				p.playPause()
			}
		}()
	}

	if p.stopBtn.Clicked() {
		if p.started {
			go func() {
				p.stop()
			}()
		}
	}

	playIcon := p.res.PlayIcon
	if p.playing {
		playIcon = p.res.PauseIcon
	}

	playBtn := func(gtx C) D {
		return material.Clickable(gtx, &p.playBtn, func(gtx C) D {
			return playIcon.Layout(gtx, unit.Dp(28))
		})
	}

	var stopBtn layout.Widget
	if p.started {
		p.res.StopIcon.Color = rgb(0)
		stopBtn = func(gtx C) D {
			return material.Clickable(gtx, &p.stopBtn, func(gtx C) D {
				return p.res.StopIcon.Layout(gtx, unit.Dp(28))
			})
		}
	} else {
		p.res.StopIcon.Color = rgb(0xd0d0d0)
		stopBtn = func(gtx C) D {
			return p.res.StopIcon.Layout(gtx, unit.Dp(28))
		}
	}

	return fn.Format(gtx, "hflex;inset(10,20,10,20)",
		fn.Child(";dir(center);inset(0,4,0,4)", playBtn),
		fn.Child(";dir(center);inset(0,4,10,4)", stopBtn),
		fn.Child("f;dir(center)", material.ProgressBar(th, p.progress).Layout),
		fn.Child(";inset(10,0,0,0);dir(center)", material.Caption(th, p.position).Layout),
	)
}

func rgb(c uint32) color.RGBA {
	return argb((0xff << 24) | c)
}

func argb(c uint32) color.RGBA {
	return color.RGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}
