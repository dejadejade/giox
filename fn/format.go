package fn

import (
	"image"
	"image/color"
	"log"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
)

type ChildSpec struct {
	style  string
	widget func(gtx C) D
}

func Child(s string, w func(gtx C) D) ChildSpec {
	return ChildSpec{style: s, widget: w}
}

func parseStyle(style string) (string, []string) {
	p := strings.IndexByte(style, '(')
	if p < 0 {
		return style, nil
	}
	name := style[:p]
	style = style[p+1:]
	p = strings.IndexByte(style, ')')
	if p < 0 {
		return name, nil
	}

	style = style[:p]
	return name, strings.Split(style, ",")
}

func atof(s string) float32 {
	f, _ := strconv.ParseFloat(s, 32)
	return float32(f)
}

func directionFor(s string) (layout.Direction, bool) {
	var d layout.Direction
	switch s {
	case "nw":
		d = layout.NW
	case "n":
		d = layout.N
	case "ne":
		d = layout.NE
	case "e":
		d = layout.E
	case "se":
		d = layout.SE
	case "s":
		d = layout.S
	case "sw":
		d = layout.SW
	case "w":
		d = layout.W
	case "center":
		d = layout.Center
	default:
		return d, false
	}

	return d, true
}

func alignmentFor(s string) (layout.Alignment, bool) {
	var a layout.Alignment
	switch s {
	case "middle":
		a = layout.Middle
	case "start":
		a = layout.Start
	case "end":
		a = layout.End
	case "baseline":
		a = layout.Baseline
	default:
		return a, false
	}

	return a, true
}

//
// inset(0,0,0,0)
// size(0,0)
// border(0,0,0,0,color)
// bkground(color)
// dir(s/n/e/w/se)
type formatter ChildSpec

func (f formatter) Layout(gtx C) D {
	style := f.style
	w := f.widget
	if w == nil {
		w = empty
	}
	if style == "" {
		return w(gtx)
	}

	var cur string
	p := strings.IndexByte(style, ';')
	if p >= 0 {
		cur = style[:p]
		w = formatter{style[p+1:], w}.Layout
	} else {
		cur = style
	}

	if cur == "" {
		return w(gtx)
	}

	name, params := parseStyle(cur)
	switch name {
	case "inset":
		if len(params) == 1 {
			return layout.UniformInset(unit.Dp(atof(params[0]))).Layout(gtx, w)
		} else if len(params) == 4 {
			return layout.Inset{Left: unit.Dp(atof(params[0])), Top: unit.Dp(atof(params[1])), Right: unit.Dp(atof(params[2])), Bottom: unit.Dp(atof(params[3]))}.Layout(gtx, w)
		}

	case "size":
		if len(params) == 2 {
			return sizeS{atof(params[0]), atof(params[1])}.Layout(gtx, w)
		}

	case "dir":
		if len(params) == 1 {
			d, _ := directionFor(params[0])
			return d.Layout(gtx, w)
		}

	case "border":
		if len(params) == 5 {
			x, _ := strconv.ParseInt(params[4], 16, 32)
			return borderS{atof(params[0]), atof(params[1]), atof(params[2]), atof(params[3]), rgb(uint32(x))}.Layout(gtx, w)
		}

	case "rounded":
		if len(params) == 1 {
			sz := gtx.Px(unit.Dp(atof(params[0])))
			cc := clipCircle{}
			return cc.Layout(gtx, func(gtx C) D {
				gtx.Constraints = layout.Exact(gtx.Constraints.Constrain(image.Point{X: sz, Y: sz}))
				return w(gtx)
			})
		}

	case "bkground":
		if len(params) == 1 {
			x, _ := strconv.ParseInt(params[0], 16, 32)
			return backgroundS{rgb(uint32(x))}.Layout(gtx, w)
		}
	}

	log.Printf("%#v, %v, %#v not handled\n", style, name, params)
	return D{}
}

func rgb(c uint32) color.RGBA {
	return argb((0xff << 24) | c)
}

func argb(c uint32) color.RGBA {
	return color.RGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}

func parseFlex(axis layout.Axis, attr []string) layout.Flex {
	f := layout.Flex{Axis: axis}
	for _, s := range attr {
		if a, ok := alignmentFor(s); ok {
			f.Alignment = a
			continue
		}
	}
	return f
}

func empty(gtx C) D {
	return D{}
}

var Empty = empty

func formatFlex(gtx C, flex layout.Flex, style string, children ...ChildSpec) D {
	var widgets []layout.FlexChild
	for _, child := range children {
		var ins string
		w := child.widget
		if w == nil {
			w = empty
		}
		p := strings.IndexByte(child.style, ';')
		if p >= 0 {
			ins = child.style[:p]
			w = formatter{child.style[p+1:], child.widget}.Layout
		}

		var c layout.FlexChild
		if ins == "" || ins[0] != 'f' {
			c = layout.Rigid(w)
		} else {
			var weight float32 = 1.0
			if _, params := parseStyle(ins); len(params) == 1 {
				weight = atof(params[0])
			}

			c = layout.Flexed(weight, w)
		}
		widgets = append(widgets, c)
	}

	return formatter{style, func(gtx C) D { return flex.Layout(gtx, widgets...) }}.Layout(gtx)
}

func parseStack(attr []string) layout.Stack {
	s := layout.Stack{}
	for _, a := range attr {
		if d, ok := directionFor(a); ok {
			s.Alignment = d
			continue
		}
	}
	return s
}

func formatStack(gtx C, stack layout.Stack, style string, children ...ChildSpec) D {
	var widgets []layout.StackChild
	for _, child := range children {
		var ins string
		w := child.widget
		if w == nil {
			w = empty
		}
		p := strings.IndexByte(child.style, ';')
		if p >= 0 {
			ins = child.style[:p]
			w = formatter{child.style[p+1:], child.widget}.Layout
		}

		var c layout.StackChild
		if ins == "" || ins[0] != 'e' {
			c = layout.Stacked(w)
		} else {
			c = layout.Expanded(w)
		}
		widgets = append(widgets, c)
	}

	return formatter{style, func(gtx C) D { return stack.Layout(gtx, widgets...) }}.Layout(gtx)
}

func Format(gtx C, style string, children ...ChildSpec) D {
	p := strings.IndexByte(style, ';')
	var name string
	var params []string
	if p > 0 {
		name, params = parseStyle(style[:p])
		style = style[p+1:]

	} else {
		name, params = parseStyle(style)
		style = ""
	}

	switch name {
	case "vflex":
		return formatFlex(gtx, parseFlex(layout.Vertical, params), style, children...)
	case "hflex":
		return formatFlex(gtx, parseFlex(layout.Horizontal, params), style, children...)
	case "stack":
		return formatStack(gtx, parseStack(params), style, children...)
	}

	log.Printf("Unhandled style: %s\n", style)
	return D{}
}

func FormatF(style string, children ...ChildSpec) layout.Widget {
	return func(gtx C) D {
		return Format(gtx, style, children...)
	}
}

func Widget(gtx C, style string, w layout.Widget) D {
	return formatter{style, w}.Layout(gtx)
}

func WidgetF(style string, w layout.Widget) layout.Widget {
	return func(gtx C) D {
		return formatter{style, w}.Layout(gtx)
	}
}
