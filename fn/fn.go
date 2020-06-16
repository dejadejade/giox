// SPDX-License-Identifier: Unlicense OR MIT

package fn

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type Style func(w layout.Widget) layout.Widget

func Styled(w layout.Widget, styles ...Style) layout.Widget {
	for i := len(styles) - 1; i >= 0; i-- {
		w = styles[i](w)
	}
	return w
}

func Size(width, height float32) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			return sizeS{width, height}.Layout(gtx, w)
		}
	}
}

type sizeS struct {
	width, height float32
}

func (s sizeS) Layout(gtx C, w layout.Widget) D {
	width, height := s.width, s.height
	cs := gtx.Constraints
	if width > 0 {
		ww := gtx.Px(unit.Dp(width))
		if ww < cs.Min.X {
			ww = cs.Min.X
		}
		if ww > cs.Max.X {
			ww = cs.Max.X
		}
		gtx.Constraints.Min.X = ww
		gtx.Constraints.Max.X = ww
	}
	if height > 0 {
		hh := gtx.Px(unit.Dp(height))
		if hh < cs.Min.Y {
			hh = cs.Min.Y
		}
		if hh > cs.Max.Y {
			hh = cs.Max.Y
		}
		gtx.Constraints.Min.Y = hh
		gtx.Constraints.Max.Y = hh
	}
	return w(gtx)
}

func drawRect(ops *op.Ops, x, y, w, h float32, fillcolor color.RGBA) {
	r := f32.Rect(x, y+h, x+w, y)
	paint.ColorOp{Color: fillcolor}.Add(ops)
	paint.PaintOp{Rect: r}.Add(ops)
}

func Border(left, top, right, bottom float32, col color.RGBA) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			return borderS{left, top, right, bottom, col}.Layout(gtx, w)
		}
	}
}

type borderS struct {
	left, top, right, bottom float32
	col                      color.RGBA
}

func (s borderS) Layout(gtx C, widget layout.Widget) D {
	m := op.Record(gtx.Ops)
	dims := widget(gtx)
	call := m.Stop()

	ops := gtx.Ops

	left, top, right, bottom := s.left, s.top, s.right, s.bottom
	col := s.col
	defer op.Push(gtx.Ops).Pop()
	w, h := float32(dims.Size.X), float32(dims.Size.Y)
	if left > 0 {
		drawRect(ops, 0, 0, left, h, col)
	}
	if top > 0 {
		drawRect(ops, 0, 0, w, top, col)
	}
	if right > 0 && w > right {
		drawRect(ops, w-right, 0, right, h, col)
	}
	if bottom > 0 && h > bottom {
		drawRect(ops, 0, h-bottom, w, bottom, col)
	}

	call.Add(gtx.Ops)
	return dims
}

func FillRect(col color.RGBA, sz image.Point) layout.Widget {
	return func(gtx C) D {
		w := gtx.Px(unit.Dp(float32(sz.X)))
		h := gtx.Px(unit.Dp(float32(sz.Y)))
		paint.ColorOp{Color: col}.Add(gtx.Ops)
		paint.PaintOp{Rect: f32.Rectangle{
			Max: f32.Point{
				X: float32(w),
				Y: float32(h),
			},
		}}.Add(gtx.Ops)
		return layout.Dimensions{
			Size: image.Point{X: w, Y: h},
		}
	}
}

func Fill(col color.RGBA) layout.Widget {
	return func(gtx C) D {
		cs := gtx.Constraints
		d := cs.Min
		dr := f32.Rectangle{
			Max: f32.Point{X: float32(d.X), Y: float32(d.Y)},
		}
		paint.ColorOp{Color: col}.Add(gtx.Ops)
		paint.PaintOp{Rect: dr}.Add(gtx.Ops)
		return layout.Dimensions{Size: d}
	}
}

type backgroundS struct {
	col color.RGBA
}

func (s backgroundS) Layout(gtx C, w layout.Widget) D {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(Fill(s.col)),
		layout.Stacked(w),
	)
}

func Rounded(r float32) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			sz := gtx.Px(unit.Dp(r))
			cc := clipCircle{}
			return cc.Layout(gtx, func(gtx C) D {
				gtx.Constraints = layout.Exact(gtx.Constraints.Constrain(image.Point{X: sz, Y: sz}))
				return w(gtx)
			})
		}
	}
}

type clipCircle struct {
}

func (c *clipCircle) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	m := op.Record(gtx.Ops)
	dims := w(gtx)
	call := m.Stop()
	max := dims.Size.X
	if dy := dims.Size.Y; dy > max {
		max = dy
	}
	szf := float32(max)
	rr := szf * .5
	defer op.Push(gtx.Ops).Pop()
	clip.Rect{
		Rect: f32.Rectangle{Max: f32.Point{X: szf, Y: szf}},
		NE:   rr, NW: rr, SE: rr, SW: rr,
	}.Op(gtx.Ops).Add(gtx.Ops)
	call.Add(gtx.Ops)
	return dims
}
