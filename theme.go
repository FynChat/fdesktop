package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Telegram's palette: one blue accent, everything else grey. Dark and light
// are the same design with different surfaces.
var (
	accent     = color.NRGBA{R: 0x33, G: 0x90, B: 0xEC, A: 0xff} // Telegram blue
	accentDark = color.NRGBA{R: 0x2B, G: 0x52, B: 0x78, A: 0xff} // own bubble, muted
	white      = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// currentIsDark tracks which theme is active. Fyne's ThemeVariant is
// read-only and follows the operating system, so a theme that must look dark
// regardless of the system preference has to say so itself; this flag is how
// the widgets that paint their own colours (message bubbles, pills) find out.
var currentIsDark = true

// telegramTheme is the dark palette. It ignores the variant it is handed and
// always renders dark, which is what makes the in-app toggle work: the app
// picks a theme rather than inheriting the system one.
type telegramTheme struct {
	fyne.Theme
}

func newTelegramTheme() fyne.Theme { return telegramTheme{Theme: theme.DefaultTheme()} }

// lightTheme is the same design on white surfaces.
type lightTheme struct {
	fyne.Theme
}

func newLightTheme() fyne.Theme { return lightTheme{Theme: theme.DefaultTheme()} }

func (t telegramTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return accent
	case theme.ColorNameForegroundOnPrimary, theme.ColorNameForegroundOnError,
		theme.ColorNameForegroundOnSuccess, theme.ColorNameForegroundOnWarning:
		return white
	case theme.ColorNameHyperlink:
		return accent
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x0E, G: 0x16, B: 0x21, A: 0xff}
	case theme.ColorNameHeaderBackground, theme.ColorNameMenuBackground:
		return color.NRGBA{R: 0x17, G: 0x21, B: 0x2B, A: 0xff}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xF2, G: 0xF5, B: 0xF8, A: 0xff}
	case theme.ColorNameInputBackground, theme.ColorNameButton:
		return color.NRGBA{R: 0x18, G: 0x25, B: 0x33, A: 0xff}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1F}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x21, G: 0x2B, B: 0x38, A: 0xff}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x33, G: 0x90, B: 0xEC, A: 0x99}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x14}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x8A, G: 0x9A, B: 0xA8, A: 0xff}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x33}
	case theme.ColorNameError:
		return color.NRGBA{R: 0xE8, G: 0x53, B: 0x4F, A: 0xff}
	}
	return t.Theme.Color(name, theme.VariantDark)
}

func (l lightTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return accent
	case theme.ColorNameForegroundOnPrimary, theme.ColorNameForegroundOnError,
		theme.ColorNameForegroundOnSuccess, theme.ColorNameForegroundOnWarning:
		return white
	case theme.ColorNameHyperlink:
		return accent
	case theme.ColorNameBackground, theme.ColorNameHeaderBackground, theme.ColorNameMenuBackground:
		return white
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0x0F, G: 0x14, B: 0x19, A: 0xff}
	case theme.ColorNameInputBackground, theme.ColorNameButton:
		return color.NRGBA{R: 0xF1, G: 0xF3, B: 0xF5, A: 0xff}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 0xD6, G: 0xD9, B: 0xDE, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0xE9, G: 0xEB, B: 0xEE, A: 0xff}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x33, G: 0x90, B: 0xEC, A: 0x55}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x14}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x70, G: 0x75, B: 0x79, A: 0xff}
	case theme.ColorNameError:
		return color.NRGBA{R: 0xD1, G: 0x3A, B: 0x37, A: 0xff}
	}
	return l.Theme.Color(name, theme.VariantLight)
}

// sizes are shared: the difference between the two themes is colour only.
func telegramSizes() fyne.Theme {
	return telegramTheme{Theme: theme.DefaultTheme()}
}

func (t telegramTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInputRadius, theme.SizeNameButtonRadius:
		return 8
	case theme.SizeNameCardRadius, theme.SizeNameDialogRadius:
		return 12
	case theme.SizeNameHeadingText:
		return 26
	case theme.SizeNameSubHeadingText:
		return 18
	case theme.SizeNameCaptionText:
		return 12
	}
	return t.Theme.Size(name)
}

func (l lightTheme) Size(name fyne.ThemeSizeName) float32 {
	return telegramSizes().Size(name)
}

// incomingBubble is the fill behind someone else's message.
func incomingBubble() color.Color {
	if currentIsDark {
		return color.NRGBA{R: 0x18, G: 0x25, B: 0x33, A: 0xff}
	}
	return color.NRGBA{R: 0xF1, G: 0xF1, B: 0xF1, A: 0xff}
}

// outgoingBubble is the fill behind your own message.
func outgoingBubble() color.Color {
	if currentIsDark {
		return accentDark
	}
	return color.NRGBA{R: 0xD6, G: 0xE9, B: 0xFC, A: 0xff}
}

// pillBackground is behind the date separators and banners.
func pillBackground() color.Color {
	if currentIsDark {
		return color.NRGBA{R: 0x1F, G: 0x2C, B: 0x3D, A: 0xff}
	}
	return color.NRGBA{R: 0xE7, G: 0xE9, B: 0xEC, A: 0xff}
}
