package main

import (
	"fmt"
	"image"
	"log"
	"sync"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dejadejade/giox/fn"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type pdfPage struct {
	doc *PDFDocument

	idx     int
	loading bool
	failed  bool
	img     image.Image
	imgOp   paint.ImageOp
}

type PDFDocument struct {
	list  *layout.List
	pages []*pdfPage
	mutex sync.Mutex

	pdoc uintptr

	notify func()
	file   string
	data   []byte
}

func NewPDFDocument(file string, notify func()) (*PDFDocument, error) {
	pdf := &PDFDocument{file: file, notify: notify, list: &layout.List{Axis: layout.Vertical, Alignment: layout.Middle}}
	return pdf, pdf.load()
}

func NewPDFDocumentWithData(data []byte, notify func()) (*PDFDocument, error) {
	pdf := &PDFDocument{data: data, notify: notify, list: &layout.List{Axis: layout.Vertical, Alignment: layout.Middle}}
	return pdf, pdf.load()
}

func (pdf *PDFDocument) Page(idx int) (*pdfPage, error) {
	if idx < 0 || idx >= len(pdf.pages) {
		return nil, fmt.Errorf("Out of range")
	}

	page := pdf.pages[idx]
	if page == nil {
		page = &pdfPage{doc: pdf, idx: idx}
		pdf.pages[idx] = page
	}
	return page, nil
}

func (pdf *PDFDocument) NumPages() int {
	return len(pdf.pages)
}

func (pdf *PDFDocument) Layout(gtx C) D {
	w, h := gtx.Constraints.Max.X, gtx.Constraints.Max.Y
	page := func(gtx C, idx int) D {
		if page, err := pdf.Page(idx); err == nil {
			return fn.Format(gtx, "hflex;dir(center);inset(4)",
				fn.Child(";inset(4);border(1,1,1,1,a0a0a0);inset(4)", func(gtx C) D {
					return page.Layout(gtx, w, h)
				}))
		}
		return D{}
	}

	npages := pdf.NumPages()
	view := func(gtx C) D {
		return pdf.list.Layout(gtx, npages, page)
	}

	return view(gtx)
}

func (page *pdfPage) Layout(gtx C, w, h int) D {
	if page.img == nil && !page.loading {
		page.loading = true
		go func() {
			var err error
			if page.img, err = page.doc.renderPage(page.idx, 160.); err != nil {
				log.Printf("Failed to render page %d: %v\n", page.idx, err)
				page.failed = true
			} else {
				page.imgOp = paint.NewImageOp(page.img)
			}
			page.loading = false
			page.doc.notify()
		}()
	}

	if page.loading || page.failed || page.img == nil {
		s := "Loading"
		if page.failed {
			s = "Failed"
		}
		s = fmt.Sprintf("Page %d: %s", page.idx, s)

		gtx.Constraints.Min.Y = h
		return material.Caption(th, s).Layout(gtx)
	}

	mw, mh := int(w/px(gtx, 1)), int(h/px(gtx, 1))
	op := page.imgOp

	var scale float32 = 1.0
	sz := op.Size()
	if sz.X > mw {
		scale = float32(mw) / float32(sz.X)
	}
	if sz.Y > mh {
		s := float32(mh) / float32(sz.Y)
		if scale > s {
			scale = s
		}
	}

	return widget.Image{Src: op, Scale: scale}.Layout(gtx)
}

func px(gtx C, x int) int {
	return gtx.Px(unit.Dp(float32(x)))
}
