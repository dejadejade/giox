// SPDX-License-Identifier: Unlicense OR MIT

package main

// A Gio program that displays Go contributors from GitHub. See https://gioui.org for more information.

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"runtime"

	"gioui.org/font/gofont"
	"gioui.org/gesture"
	// "gioui.org/io/pointer"
	"gioui.org/io/profile"
	"gioui.org/layout"
	"gioui.org/op/paint"
	//"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/google/go-github/v24/github"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"golang.org/x/image/draw"

	"github.com/dejadejade/giox/fn"
)

type UI struct {
	fab          *widget.Clickable
	fabIcon      *widget.Icon
	fabIcon2     *widget.Icon
	usersList    *layout.List
	users        []*user
	userClicks   []gesture.Click
	selectedUser *userPage
	edit, edit2  *widget.Editor
	fetchCommits func(u string)

	// Profiling.
	profiling   bool
	profile     profile.Event
	lastMallocs uint64
}

type userPage struct {
	user        *user
	commitsList *layout.List
	commits     []*github.Commit
}

type user struct {
	name     string
	login    string
	company  string
	avatar   image.Image
	avatarOp paint.ImageOp
}

var theme *material.Theme

type (
	C = layout.Context
	D = layout.Dimensions
	S = fn.Style
)

func init() {
	gofont.Register()
	theme = material.NewTheme()
	theme.Color.Text = rgb(0x333333)
	theme.Color.Hint = rgb(0xbbbbbb)
}

func newUI(fetchCommits func(string)) *UI {
	u := &UI{
		fetchCommits: fetchCommits,
	}
	u.usersList = &layout.List{
		Axis: layout.Vertical,
	}
	u.fab = new(widget.Clickable)
	u.edit2 = &widget.Editor{
		//Alignment: text.End,
		SingleLine: true,
	}
	var err error
	u.fabIcon, err = widget.NewIcon(icons.ContentSend)
	if err != nil {
		log.Fatal(err)
	}
	u.fabIcon2, err = widget.NewIcon(icons.NavigationArrowBack)
	if err != nil {
		log.Fatal(err)
	}

	u.edit2.SetText("Single line editor. Edit me!")
	u.edit2.SetText("Single line editor. Edit me!")
	u.edit = &widget.Editor{
		//Alignment: text.End,
		//SingleLine: true,
	}
	u.edit.SetText(longTextSample)
	return u
}

func rgb(c uint32) color.RGBA {
	return argb((0xff << 24) | c)
}

func argb(c uint32) color.RGBA {
	return color.RGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}

func (u *UI) layoutTimings(gtx layout.Context) string {
	for _, e := range gtx.Events(u) {
		if e, ok := e.(profile.Event); ok {
			u.profile = e
		}
	}
	profile.Op{Tag: u}.Add(gtx.Ops)

	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)
	mallocs := mstats.Mallocs - u.lastMallocs
	u.lastMallocs = mstats.Mallocs

	txt := fmt.Sprintf("m: %d %s", mallocs, u.profile.Timings)
	return txt
}

func (u *UI) newUserPage(user *user) *userPage {
	up := &userPage{
		user:        user,
		commitsList: &layout.List{Axis: layout.Vertical},
	}
	u.fetchCommits(user.login)
	return up
}

func (u *UI) Layout(gtx layout.Context) {
	for i := range u.userClicks {
		click := &u.userClicks[i]
		for _, e := range click.Events(gtx) {
			if e.Type == gesture.TypeClick {
				u.selectedUser = u.newUserPage(u.users[i])
			}
		}
	}

	if u.fab.Clicked() {
		if u.selectedUser != nil {
			u.selectedUser = nil
		} else {
			u.users = append([]*user{u.users[1], u.users[2]}, u.users...)
			u.usersList.Position.First += 2
		}
	}

	if u.selectedUser != nil {
		UserPage(u)(gtx)
	} else {
		Users(u)(gtx)
	}

	if u.profiling {
		txt := u.layoutTimings(gtx)
		fn.Styled(material.Caption(theme, txt).Layout, fn.Direction(layout.NE), fn.Margin4(0, 16, 0, 0))(gtx)
	}
}

func UserPage(u *UI) layout.Widget {
	up := u.selectedUser
	content := fn.List(up.commitsList, nil, len(up.commits), func(gtx C, i int) D {
		return Commit(up.user, up.commits[i].GetMessage())(gtx)
	})

	main := fn.Stack(layout.Stack{Alignment: layout.SE}, nil,
		fn.StackChild{fn.Expanded(false), nil, content},
		fn.StackChild{fn.Stacked(), []S{fn.Margin(16)}, material.IconButton(theme, u.fab, u.fabIcon2).Layout},
	)

	return main
}

func Commit(user *user, msg string) layout.Widget {
	return fn.Flex(layout.Flex{Axis: layout.Horizontal}, []S{fn.Margin4(8, 16, 8, 0)},
		fn.FlexChild{fn.Rigid(false), []S{fn.Rounded(48)}, Avatar(user)},
		fn.FlexChild{fn.Flexed(1), []S{fn.Margin4(8, 0, 0, 0)}, fn.Label(material.Caption(theme, msg))},
	)
}

func User(user *user, click *gesture.Click) layout.Widget {
	content := fn.Flex(layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}, []S{fn.Margin(8), fn.OnClick(click)},
		fn.FlexChild{fn.Rigid(false), []S{fn.Margin(8), fn.Rounded(36)}, Avatar(user)},
		fn.FlexChild{fn.Rigid(false), nil,
			fn.Flex(layout.Flex{Axis: layout.Vertical}, nil,
				fn.FlexChild{fn.Rigid(false), nil,
					fn.Flex(layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}, nil,
						fn.FlexChild{fn.Rigid(false), nil, material.Body1(theme, user.name).Layout},
						fn.FlexChild{fn.Flexed(1), []S{fn.Direction(layout.E), fn.Margin4(2, 0, 0, 0)}, material.Caption(theme, "3 hours ago").Layout},
					)},
				fn.FlexChild{fn.Rigid(false), []S{fn.Margin4(0, 4, 0, 0)}, material.Caption(theme, user.company).Layout},
			)},
	)

	return content
}

func Users(u *UI) layout.Widget {
	content := fn.Flex(layout.Flex{Axis: layout.Vertical}, nil,
		fn.FlexChild{fn.Rigid(true), []S{fn.Margin(16), fn.Size(400, 200)}, material.Editor(theme, u.edit, "Hint").Layout},
		fn.FlexChild{fn.Rigid(true), []S{fn.Margin(16)}, material.Editor(theme, u.edit2, "Hint").Layout},
		fn.FlexChild{fn.Rigid(false), []S{fn.Background(rgb(0xf2f2f2))}, fn.Styled(material.Caption(theme, "GOPHERS").Layout, fn.Margin(8))},
		fn.FlexChild{fn.Flexed(1), nil,
			fn.List(u.usersList, nil, len(u.users), func(gtx C, index int) D {
				return User(u.users[index], &u.userClicks[index])(gtx)
			})},
	)

	main := fn.Stack(layout.Stack{Alignment: layout.SE}, nil,
		fn.StackChild{fn.Expanded(false), nil, content},
		fn.StackChild{fn.Stacked(), []S{fn.Margin(16)}, material.IconButton(theme, u.fab, u.fabIcon).Layout},
	)

	return main
}

func Avatar(u *user) layout.Widget {
	return func(gtx C) D {
		sz := gtx.Constraints.Min.X
		if u.avatarOp.Size().X != sz {
			img := image.NewRGBA(image.Rectangle{Max: image.Point{X: sz, Y: sz}})
			draw.ApproxBiLinear.Scale(img, img.Bounds(), u.avatar, u.avatar.Bounds(), draw.Src, nil)
			u.avatarOp = paint.NewImageOp(img)
		}
		img := widget.Image{Src: u.avatarOp}
		img.Scale = float32(sz) / float32(gtx.Px(unit.Dp(float32(sz))))
		return img.Layout(gtx)
	}
}

const longTextSample = `1. I learned from my grandfather, Verus, to use good manners, and to
put restraint on anger. 2. In the famous memory of my father I had a
pattern of modesty and manliness. 3. Of my mother I learned to be
pious and generous; to keep myself not only from evil deeds, but even
from evil thoughts; and to live with a simplicity which is far from
customary among the rich. 4. I owe it to my great-grandfather that I
did not attend public lectures and discussions, but had good and able
teachers at home; and I owe him also the knowledge that for things of
this nature a man should count no expense too great.

5. My tutor taught me not to favour either green or blue at the
chariot races, nor, in the contests of gladiators, to be a supporter
either of light or heavy armed. He taught me also to endure labour;
not to need many things; to serve myself without troubling others; not
to intermeddle in the affairs of others, and not easily to listen to
slanders against them.

6. Of Diognetus I had the lesson not to busy myself about vain things;
not to credit the great professions of such as pretend to work
wonders, or of sorcerers about their charms, and their expelling of
Demons and the like; not to keep quails (for fighting or divination),
nor to run after such things; to suffer freedom of speech in others,
and to apply myself heartily to philosophy. Him also I must thank for
my hearing first Bacchius, then Tandasis and Marcianus; that I wrote
dialogues in my youth, and took a liking to the philosopher's pallet
and skins, and to the other things which, by the Grecian discipline,
belong to that profession.

7. To Rusticus I owe my first apprehensions that my nature needed
reform and cure; and that I did not fall into the ambition of the
common Sophists, either by composing speculative writings or by
declaiming harangues of exhortation in public; further, that I never
strove to be admired by ostentation of great patience in an ascetic
life, or by display of activity and application; that I gave over the
study of rhetoric, poetry, and the graces of language; and that I did
not pace my house in my senatorial robes, or practise any similar
affectation. I observed also the simplicity of style in his letters,
particularly in that which he wrote to my mother from Sinuessa. I
learned from him to be easily appeased, and to be readily reconciled
with those who had displeased me or given cause of offence, so soon as
they inclined to make their peace; to read with care; not to rest
satisfied with a slight and superficial knowledge; nor quickly to
assent to great talkers. I have him to thank that I met with the
discourses of Epictetus, which he furnished me from his own library.

8. From Apollonius I learned true liberty, and tenacity of purpose; to
regard nothing else, even in the smallest degree, but reason always;
and always to remain unaltered in the agonies of pain, in the losses
of children, or in long diseases. He afforded me a living example of
how the same man can, upon occasion, be most yielding and most
inflexible. He was patient in exposition; and, as might well be seen,
esteemed his fine skill and ability in teaching others the principles
of philosophy as the least of his endowments. It was from him that I
learned how to receive from friends what are thought favours without
seeming humbled by the giver or insensible to the gift.`
