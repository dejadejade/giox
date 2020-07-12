package player

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"time"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/dejadejade/giox/fn"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// AudioPlayerResource holds the widget resources
type PlayerResource struct {
	PlayIcon  *widget.Icon
	StopIcon  *widget.Icon
	PauseIcon *widget.Icon
	Theme     *material.Theme
}

type command uint8

const (
	cmdPause = iota
	cmdPlay
)

func rgb(c uint32) color.RGBA {
	return argb((0xff << 24) | c)
}

func argb(c uint32) color.RGBA {
	return color.RGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}

func format(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

type Picture struct {
	img image.Image
	pts float64
}

type AudioSample struct {
	data       []byte
	pts        float64
	numSamples int
}

type VideoPlayer struct {
	playBtn widget.Clickable
	stopBtn widget.Clickable

	res    *PlayerResource
	driver *driver

	started bool
	playing bool

	duration time.Duration
	progress int
	position string

	notify  chan func()
	picture Picture
	// Output  chan Picture
	// picture atomic.Value
}

func NewVideoPlayer(file string, res *PlayerResource, notify chan func()) *VideoPlayer {
	//	output := make(chan Picture)
	if res == nil {
		th := material.NewTheme(gofont.Collection())
		stopIcon, _ := widget.NewIcon(icons.AVStop)
		playIcon, _ := widget.NewIcon(icons.AVPlayArrow)
		pauseIcon, _ := widget.NewIcon(icons.AVPause)
		res = &PlayerResource{PlayIcon: playIcon, PauseIcon: pauseIcon, StopIcon: stopIcon, Theme: th}
	}

	vp := &VideoPlayer{res: res, position: "00:00:00", notify: notify}
	onPicture := func(pic Picture) {
		vp.notify <- func() {
			vp.onPicture(pic)
		}
	}
	onStartStop := func(start bool) {
		vp.notify <- func() {
			vp.onStartStop(start)
		}
	}
	vp.driver = newDriver(file, onPicture, onStartStop)
	return vp
}

func (p *VideoPlayer) Start() error {
	err := p.driver.start()
	if err != nil {
		return err
	}

	return nil
}

func (p *VideoPlayer) Started() bool {
	return p.started
}

func (p *VideoPlayer) onStartStop(start bool) {
	if start {
		p.duration = time.Duration(time.Millisecond * time.Duration(p.driver.duration/1000))
		p.position = format(p.duration)
		p.started = true
		p.playing = true
	} else {
		p.playing = false
		p.started = false
		p.duration = time.Duration(0)
		p.progress = 0
		p.position = "00:00:00"
		p.picture = Picture{}
	}
}

func (p *VideoPlayer) playPause() {
	p.playing = !p.playing
	if p.playing {
		p.driver.done <- cmdPlay
	} else {
		p.driver.done <- cmdPause
	}
}

func (p *VideoPlayer) onPicture(pic Picture) {
	p.picture = pic
	x := time.Duration(pic.pts+.5) * time.Second
	p.position = format(p.duration - x)
	p.progress = int(x.Seconds()) * 100 / int(p.duration.Seconds())
}

func (p *VideoPlayer) layoutControls(gtx C) D {
	th := p.res.Theme
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

	return fn.Format(gtx, "hflex;inset(5,0,5,0)",
		fn.Child(";dir(center)", playBtn),
		fn.Child(";dir(center)", stopBtn),
		fn.Child("f;inset(0,0,10,0);dir(center)", material.ProgressBar(th, p.progress).Layout),
		fn.Child(";inset(10,0,0,0);dir(center)", material.Caption(th, p.position).Layout),
	)
}

func (p *VideoPlayer) Layout(gtx C) D {
	if p.playBtn.Clicked() {
		if !p.started {
			go func() {
				if err := p.Start(); err != nil {
					log.Printf("Failed to start: %v\n", err)
				}
			}()
		} else {
			go p.playPause()
		}
	}

	if p.stopBtn.Clicked() {
		if p.started {
			go p.driver.stop()
		}
	}

	var children []fn.ChildSpec

	if p.picture.img != nil {
		children = append(children, fn.Child("f;dir(center)", func(gtx C) D {
			mw, mh := gtx.Constraints.Max.X, gtx.Constraints.Max.Y
			op := paint.NewImageOp(p.picture.img)
			sz := op.Size()
			scale := float32(mw) / float32(gtx.Px(unit.Dp(1))) / float32(sz.X)
			s := float32(mh) / float32(sz.Y)
			if scale > s {
				scale = s
			}
			return widget.Image{Src: op, Scale: scale}.Layout(gtx)
		}))
	}
	children = append(children, fn.Child(";border(0,0,0,1,c0c0c0);size(0,50)", func(gtx C) D {
		return p.layoutControls(gtx)
	}))

	return fn.Format(gtx, "vflex;", children...)
}
