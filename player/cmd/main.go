package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/dejadejade/giox/player"
)

var ap *player.VideoPlayer

var notify chan func()

func main() {
	player.InitDriver()

	notify = make(chan func())
	ap = player.NewVideoPlayer(os.Args[1], nil, notify)

	go func() {
		defer os.Exit(0)
		w := app.NewWindow(app.Title("Player"), app.Size(unit.Dp(800), unit.Dp(600)))
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
	}()
	//	ap.Start()
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
		case cb := <-notify:
			{
				if cb != nil {
					cb()
				}
				w.Invalidate()
			}
		}
	}
}
