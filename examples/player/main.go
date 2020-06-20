package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var ap *AudioPlayer

var th *material.Theme

func main() {
	th = material.NewTheme(gofont.Collection())
	w := app.NewWindow(app.Title("Player"), app.Size(unit.Dp(400), unit.Dp(80)))
	if len(os.Args) < 2 {
		log.Printf("Usage: player xxx.mp3")
		os.Exit(0)
	}

	stopIcon, _ := widget.NewIcon(icons.AVStop)
	playIcon, _ := widget.NewIcon(icons.AVPlayArrow)
	pauseIcon, _ := widget.NewIcon(icons.AVPause)

	ap = NewAudioPlayer(os.Args[1], &AudioPlayerResource{PlayIcon: playIcon, PauseIcon: pauseIcon, StopIcon: stopIcon}, func() {
		w.Invalidate()
	})

	go func() {
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
	}()
	app.Main()
}

func loop(w *app.Window) error {
	var ops op.Ops
	for {
		select {
		case e := <-w.Events():
			switch e := e.(type) {
			case system.DestroyEvent:
				return e.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				ap.Layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}
}
