// SPDX-License-Identifier: Unlicense OR MIT

package fn

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	//	"gioui.org/widget"
	"gioui.org/widget/material"
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

func Margin(x float32) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			return layout.UniformInset(unit.Dp(x)).Layout(gtx, w)
		}
	}
}

func Margin4(left, top, right, bottom float32) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			in := layout.Inset{Top: unit.Dp(top), Right: unit.Dp(right), Left: unit.Dp(left), Bottom: unit.Dp(bottom)}
			return in.Layout(gtx, w)
		}
	}
}

func Size(width, height float32) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			ww, hh := gtx.Px(unit.Dp(width)), gtx.Px(unit.Dp(height))
			cs := gtx.Constraints
			gtx.Constraints = layout.Exact(cs.Constrain(image.Point{X: ww, Y: hh}))
			return w(gtx)
		}
	}
}

func Direction(d layout.Direction) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			return d.Layout(gtx, w)
		}
	}
}

func drawRect(ops *op.Ops, x, y, w, h float32, fillcolor color.RGBA) {
	r := f32.Rect(x, y+h, x+w, y)
	paint.ColorOp{Color: fillcolor}.Add(ops)
	paint.PaintOp{Rect: r}.Add(ops)
}

func Border(left, top, right, bottom float32, col color.RGBA) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			m := op.Record(gtx.Ops)
			dims := w(gtx)
			call := m.Stop()

			ops := gtx.Ops

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
	}
}

func Visible(v bool) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			if !v {
				return layout.Dimensions{}
			}
			return w(gtx)
		}
	}
}

func OnClick(click *gesture.Click) Style {
	return func(w layout.Widget) layout.Widget {
		return func(gtx C) D {
			dims := w(gtx)
			pointer.Rect(image.Rectangle{Max: dims.Size}).Add(gtx.Ops)
			click.Add(gtx.Ops)
			return dims
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

func Background(col color.RGBA) Style {
	return func(w layout.Widget) layout.Widget {
		return Stack(layout.Stack{}, nil,
			StackChild{Expanded(true), nil, Fill(col)},
			StackChild{Stacked(), nil, w},
		)
	}
}

func Rigid(expand bool) FlexChildFunc {
	return func(w layout.Widget) layout.FlexChild {
		return layout.Rigid(func(gtx C) D {
			if expand {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
			}
			return w(gtx)
		})
	}
}

func Flexed(i float32) FlexChildFunc {
	return func(w layout.Widget) layout.FlexChild {
		return layout.Flexed(i, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return w(gtx)
		})
	}
}

type FlexChildFunc func(w layout.Widget) layout.FlexChild
type FlexChild struct {
	Wrapper FlexChildFunc
	Styles  []Style
	Widget  layout.Widget
}

type StackChildFunc func(w layout.Widget) layout.StackChild
type StackChild struct {
	Wrapper StackChildFunc
	Styles  []Style
	Widget  layout.Widget
}

func Flex(flex layout.Flex, styles []Style, children ...FlexChild) layout.Widget {
	w := func(gtx C) D {
		var widgets []layout.FlexChild
		for _, child := range children {
			widget := child.Wrapper(Styled(child.Widget, child.Styles...))
			widgets = append(widgets, widget)
		}
		return flex.Layout(gtx, widgets...)
	}

	return Styled(w, styles...)
}

func Stack(stack layout.Stack, styles []Style, children ...StackChild) layout.Widget {
	w := func(gtx C) D {
		var widgets []layout.StackChild
		for _, child := range children {
			widget := child.Wrapper(Styled(child.Widget, child.Styles...))
			widgets = append(widgets, widget)
		}
		return stack.Layout(gtx, widgets...)
	}

	return Styled(w, styles...)
}

func Expanded(expand bool) StackChildFunc {
	return func(w layout.Widget) layout.StackChild {
		return layout.Expanded(func(gtx C) D {
			if expand {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
			}
			return w(gtx)
		})
	}
}

func Stacked() StackChildFunc {
	return func(w layout.Widget) layout.StackChild {
		return layout.Stacked(func(gtx C) D {
			return w(gtx)
		})
	}
}

func List(list *layout.List, styles []Style, n int, child layout.ListElement) layout.Widget {
	w := func(gtx C) D {
		if list.Dragging() {
			key.HideInputOp{}.Add(gtx.Ops)
		}
		return list.Layout(gtx, n, child)
	}

	return Styled(w, styles...)
}

func Label(l material.LabelStyle) layout.Widget {
	return l.Layout
}

type LabelOption func(l material.LabelStyle)

func LabelFont() LabelOption {
	return func(l material.LabelStyle) {
	}
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
