package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// kindIcon marks the kind of channel at a glance: two heads for a group, a
// broadcast arrow for a channel, and nothing for a one-to-one conversation,
// where the person's own avatar says it all.
//
// Telegram does it with a small badge on the avatar; either way the name of
// the type in the row was noise.
func kindIcon(channelType string) fyne.CanvasObject {
	switch strings.ToLower(strings.TrimSpace(channelType)) {
	case "group":
		return groupIcon(16)
	case "channel":
		return broadcastIcon(16)
	default:
		return nil
	}
}

// groupIcon is two overlapping heads, drawn rather than shipped as an asset.
func groupIcon(size float32) fyne.CanvasObject {
	fill := theme.PlaceHolderColor()

	head := func(d float32) fyne.CanvasObject {
		r := canvas.NewRectangle(fill)
		r.CornerRadius = d / 2 // a rounded square of radius d/2 reads as a circle
		r.Resize(fyne.NewSize(d, d))
		return r
	}

	// WithoutLayout, because the two heads overlap on purpose and a layout
	// would keep them apart. The GridWrap wrapper is what gives the box its
	// size in the list row — dropping it leaves the heads unlaid out.
	box := container.NewWithoutLayout(head(size*0.62), head(size*0.44))
	big := box.Objects[0]
	small := box.Objects[1]
	big.Resize(fyne.NewSquareSize(size * 0.62))
	big.Move(fyne.NewPos(0, size*0.38))
	small.Resize(fyne.NewSquareSize(size * 0.44))
	small.Move(fyne.NewPos(size*0.52, size*0.16))

	return container.NewGridWrap(fyne.NewSquareSize(size), box)
}

// broadcastIcon is a megaphone drawn as a pentagon: a horn opening to the
// right, with a small block behind it.
func broadcastIcon(size float32) fyne.CanvasObject {
	fill := theme.PlaceHolderColor()

	horn := canvas.NewArbitraryPolygon([]fyne.Position{
		{X: 0, Y: size * 0.18},
		{X: size * 0.45, Y: size * 0.18},
		{X: size, Y: size * 0.5},
		{X: size * 0.45, Y: size * 0.82},
		{X: 0, Y: size * 0.82},
	}, fill)

	return container.NewGridWrap(fyne.NewSquareSize(size), horn)
}

// menuItem is one row of the side menu: an icon, a label and an optional note
// on the right, highlighted on hover like the rows in Telegram's menu.
type menuItem struct {
	widget.BaseWidget
	background *canvas.Rectangle
	row        *fyne.Container
	label      *canvas.Text
	note       *canvas.Text
	onTap      func()
}

func newMenuItem(icon fyne.Resource, label, note string, onTap func()) *menuItem {
	m := &menuItem{onTap: onTap}

	m.background = canvas.NewRectangle(color.Transparent)
	m.background.CornerRadius = 8

	m.label = canvas.NewText(label, theme.ForegroundColor())
	m.label.TextStyle = fyne.TextStyle{Bold: true}

	m.note = canvas.NewText(note, theme.PlaceHolderColor())
	m.note.TextSize = theme.Size(theme.SizeNameCaptionText)
	if note == "" {
		m.note.Hide()
	}

	// A row without an icon keeps its text aligned by reserving the same
	// space for the missing one.
	leading := fyne.CanvasObject(container.NewGridWrap(fyne.NewSquareSize(20),
		canvas.NewRectangle(color.Transparent)))
	if icon != nil {
		img := canvas.NewImageFromResource(icon)
		img.SetMinSize(fyne.NewSquareSize(20))
		leading = img
	}

	m.row = container.NewStack(m.background, container.NewPadded(container.NewBorder(nil, nil,
		container.NewHBox(leading, m.label), m.note, nil)))

	m.ExtendBaseWidget(m)
	return m
}

func (m *menuItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.row)
}

func (m *menuItem) Tapped(*fyne.PointEvent) {
	if m.onTap != nil {
		m.onTap()
	}
}

func (m *menuItem) CursorEntered(*fyne.PointEvent) {
	m.background.FillColor = hoverColour()
	m.Refresh()
}

func (m *menuItem) CursorExited(*fyne.PointEvent) {
	m.background.FillColor = color.Transparent
	m.Refresh()
}

// SetNote updates the text on the right, for the theme toggle to show what is
// currently selected.
func (m *menuItem) SetNote(note string) {
	m.note.Text = note
	if note == "" {
		m.note.Show()
	} else {
		m.note.Show()
	}
	m.Refresh()
}

// gap is vertical breathing room. A layout.NewSpacer in a VBox does nothing:
// a VBox hands every child its minimum height and never distributes the rest,
// so the space between two fields has to be asked for explicitly.
func gap(height float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(0, height))
	return r
}

// hoverColour is the row highlight, matched to whichever theme is active.
func hoverColour() color.Color {
	if currentIsDark {
		return color.NRGBA{R: 0x21, G: 0x2B, B: 0x38, A: 0xff}
	}
	return color.NRGBA{R: 0xE9, G: 0xEB, B: 0xEE, A: 0xff}
}

// sectionLabel is a small grey heading inside the side menu.
func sectionLabel(text string) fyne.CanvasObject {
	l := canvas.NewText(text, theme.PlaceHolderColor())
	l.TextSize = theme.Size(theme.SizeNameCaptionText)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewPadded(l)
}

// rightAligned pushes a widget to the right edge of its row.
func rightAligned(obj fyne.CanvasObject) fyne.CanvasObject {
	return container.NewHBox(layout.NewSpacer(), obj)
}
