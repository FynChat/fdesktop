package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

//go:embed Icon.png
var iconBytes []byte

// appIcon is the FynChat logo, embedded so the binary carries it and the
// window shows it in the title bar and the task switcher.
func appIcon() fyne.Resource {
	return fyne.NewStaticResource("Icon.png", iconBytes)
}

// appLogo draws the logo on a rounded accent tile. The source PNG has a flat
// dark square behind it, which looks like a placeholder on a dark window, so
// the pixels that match the corner colour are made transparent first and the
// logo is drawn on a tile we control.
func appLogo(size float32) fyne.CanvasObject {
	tile := canvas.NewRectangle(accent)
	tile.CornerRadius = size * 0.22
	tile.Resize(fyne.NewSize(size, size))
	tile.SetMinSize(fyne.NewSize(size, size))

	mark := canvas.NewImageFromResource(transparentLogo())
	mark.ScaleMode = canvas.ImageScalePixels
	mark.Resize(fyne.NewSize(size*0.72, size*0.72))
	mark.SetMinSize(fyne.NewSize(size*0.72, size*0.72))

	return container.NewStack(tile, container.NewCenter(mark))
}

// transparentLogo returns the logo with its flat background keyed out.
// Keying on the corner colour avoids having to know the exact shade, and
// only fully-opaque pixels are touched so anti-aliased edges stay smooth.
func transparentLogo() fyne.Resource {
	if keyed := keyedLogo(); keyed != nil {
		return keyed
	}
	return appIcon()
}

var keyedLogoResource fyne.Resource

func keyedLogo() fyne.Resource {
	if keyedLogoResource != nil {
		return keyedLogoResource
	}

	src, _, err := image.Decode(bytes.NewReader(iconBytes))
	if err != nil {
		return nil
	}
	b := src.Bounds()
	if b.Empty() {
		return nil
	}

	corner := color.NRGBAModel.Convert(src.At(b.Min.X, b.Min.Y)).(color.NRGBA)
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			// Only key out pixels that are opaque and almost exactly the
			// background shade; anything else is part of the mark.
			if p.A == 0xff && near(p, corner, 12) {
				p.A = 0
			}
			dst.SetNRGBA(x, y, p)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil
	}

	keyedLogoResource = fyne.NewStaticResource("Icon-keyed.png", buf.Bytes())
	return keyedLogoResource
}

// near reports whether two colours are within tolerance on every channel.
func near(a, b color.NRGBA, tolerance int) bool {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	return dr*dr+dg*dg+db*db <= tolerance*tolerance*3
}

// linkLabel is a tappable, accent-coloured piece of text. Fyne's widget.Hyperlink
// is only for real URLs, but "Create one" / "Sign in" are in-app navigation,
// so this stands in for the blue links Telegram uses under its forms.
type linkLabel struct {
	widget.BaseWidget
	text  *canvas.Text
	onTap func()
}

func newLinkLabel(text string, onTap func()) *linkLabel {
	l := &linkLabel{
		text:  canvas.NewText(text, theme.HyperlinkColor()),
		onTap: onTap,
	}
	l.text.TextStyle = fyne.TextStyle{Italic: true}
	l.ExtendBaseWidget(l)
	return l
}

func (l *linkLabel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.text)
}

func (l *linkLabel) Tapped(*fyne.PointEvent) {
	if l.onTap != nil {
		l.onTap()
	}
}

// initial returns the first letter of a name, uppercased, or "?" when the
// name is empty.
func initial(name string) string {
	for _, r := range strings.TrimSpace(name) {
		return strings.ToUpper(string(r))
	}
	return "?"
}

// avatar draws a coloured circle with a name's initial inside it — the same
// trick Telegram uses when a user has no profile picture.
func avatar(name string, size float32) fyne.CanvasObject {
	letter := initial(name)
	box := fyne.NewSize(size, size)

	// A rectangle with a corner radius of half its size is a circle, and it
	// supports SetMinSize, which canvas.Circle does not.
	circle := canvas.NewRectangle(avatarColor(letter))
	circle.CornerRadius = size / 2
	circle.Resize(box)
	circle.SetMinSize(box)

	text := canvas.NewText(letter, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	text.TextStyle = fyne.TextStyle{Bold: true}
	text.Alignment = fyne.TextAlignCenter
	text.Resize(box)
	text.SetMinSize(box)

	return container.NewStack(circle, text)
}

// avatarColor gives each initial a stable colour, so the same person keeps
// the same bubble colour between sessions.
func avatarColor(letter string) color.NRGBA {
	palette := []color.NRGBA{
		{R: 0xE5, G: 0x73, B: 0x73, A: 0xff},
		{R: 0xF0, G: 0x62, B: 0x92, A: 0xff},
		{R: 0xBA, G: 0x68, B: 0xC8, A: 0xff},
		{R: 0x95, G: 0x75, B: 0xCD, A: 0xff},
		{R: 0x79, G: 0x86, B: 0xCB, A: 0xff},
		{R: 0x64, G: 0xB5, B: 0xF6, A: 0xff},
		{R: 0x4F, G: 0xC3, B: 0xF7, A: 0xff},
		{R: 0x4D, G: 0xD0, B: 0xE1, A: 0xff},
		{R: 0x4D, G: 0xB6, B: 0xAC, A: 0xff},
		{R: 0x81, G: 0xC7, B: 0x84, A: 0xff},
		{R: 0xFF, G: 0xD5, B: 0x4F, A: 0xff},
		{R: 0xFF, G: 0x8A, B: 0x65, A: 0xff},
	}

	sum := 0
	for _, r := range letter {
		sum += int(r)
	}
	return palette[sum%len(palette)]
}

// headerBar is the strip at the top of the channels and chat views, with any
// trailing widgets pinned to the right.
func headerBar(title string, trailing ...fyne.CanvasObject) fyne.CanvasObject {
	titleText := canvas.NewText(title, theme.ForegroundColor())
	titleText.TextStyle = fyne.TextStyle{Bold: true}

	// Border layout, not an HBox: it stretches the title and pins the buttons
	// to the right edge whatever the window width is.
	return container.NewPadded(
		container.NewBorder(nil, nil, titleText, container.NewHBox(trailing...), nil),
	)
}

// quietButton is a low-emphasis button for actions that sit beside the real
// call to action. It is used by the auth screens, where a labelled button
// reads better than an icon.
func quietButton(label string, onTapped func()) *widget.Button {
	b := widget.NewButton(label, onTapped)
	b.Importance = widget.LowImportance
	return b
}

// iconButton is a borderless button showing a single icon, for header actions.
func iconButton(icon fyne.Resource, onTapped func()) *widget.Button {
	b := widget.NewButtonWithIcon("", icon, onTapped)
	b.Importance = widget.LowImportance
	return b
}

// asyncButton runs an action on a goroutine while the button shows a working
// label and refuses further taps. Without this, a double click on a slow
// connection sends the request twice.
func asyncButton(label, busyLabel string, run func()) *widget.Button {
	b := widget.NewButton(label, nil)
	b.OnTapped = func() {
		if !b.Disabled() {
			return
		}
		b.Disable()
		b.SetText(busyLabel)

		go func() {
			run()

			fyne.Do(func() {
				b.Enable()
				b.SetText(label)
			})
		}()
	}
	return b
}

// errorBanner is the non-blocking error strip. A modal dialog for a failed
// send is heavy: the banner sits under the content, can be dismissed, and
// leaves the form filled in.
type errorBanner struct {
	label *widget.Label
	box   *fyne.Container
	wrap  *fyne.Container
}

func errorBannerWidget() *errorBanner {
	label := widget.NewLabel("")
	label.Wrapping = fyne.TextWrapWord
	b := &errorBanner{label: label}

	background := canvas.NewRectangle(color.NRGBA{R: 0x7A, G: 0x27, B: 0x2E, A: 0xFF})
	background.CornerRadius = 8

	content := container.NewStack(background, container.NewCenter(container.NewPadded(label)))
	b.box = content
	b.wrap = container.NewCenter(content)

	b.label.Importance = widget.HighImportance
	b.clear()
	return b
}

// Show puts a message in the banner.
func (b *errorBanner) Show(err error) {
	if err == nil {
		b.clear()
		return
	}
	b.label.SetText(err.Error())
	b.show()
}

// ShowMessage puts a plain string in the banner.
func (b *errorBanner) ShowMessage(text string) {
	b.label.SetText(text)
	b.show()
}

// Clear empties the banner.
func (b *errorBanner) Clear() { b.clear() }

func (b *errorBanner) clear() {
	b.label.SetText("")
	b.wrap.Hide()
}

func (b *errorBanner) show() {
	b.wrap.Show()
	b.wrap.Refresh()
}
