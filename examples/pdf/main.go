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
	"gioui.org/widget/material"
)

var ap *PDFDocument
var th *material.Theme

func main() {
	if len(os.Args) != 2 {
		log.Printf("Usage: %s file.pdf\n", os.Args[0])
		os.Exit(0)
	}

	InitPDFLibrary()

	th = material.NewTheme(gofont.Collection())
	w := app.NewWindow(app.Title("Player"), app.Size(unit.Dp(1600), unit.Dp(1200)))

	ap, _ = NewPDFDocument(os.Args[1], func() {
		w.Invalidate()
	})

	go func() {
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
	}()
	app.Main()

	DestroyPDFLibrary()
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
