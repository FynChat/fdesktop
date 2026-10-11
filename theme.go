package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Telegram-inspired palette. Telegram keeps a single blue accent and lets
// everything else be greys, which is what makes its UI read as calm.
var (
	accent      = color.NRGBA{R: 0x33, G: 0x90, B: 0xEC, A: 0xff} // Telegram blue
	accentDark  = color.NRGBA{R: 0x2B, G: 0x52, B: 0x78, A: 0xff} // own bubble, muted
	darkBg      = color.NRGBA{R: 0x0E, G: 0x16, B: 0x21, A: 0xff} // window
	darkPanel   = color.NRGBA{R: 0x17, G: 0x21, B: 0x2B, A: 0xff} // headers, sidebars
	darkInput   = color.NRGBA{R: 0x18, G: 0x25, B: 0x33, A: 0xff}
	darkBubble  = color.NRGBA{R: 0x18, G: 0x25, B: 0x33, A: 0xff} // incoming
	darkHover   = color.NRGBA{R: 0x21, G: 0x2B, B: 0x38, A: 0xff}
	lightBubble = color.NRGBA{R: 0xF1, G: 0xF1, B: 0xF1, A: 0xff}
)

// telegramTheme customises the default Fyne theme with the palette above.
// Anything not listed here falls through to the Fyne default.
type telegramTheme struct {
	fyne.Theme
}

// newTelegramTheme returns the app theme, wrapping Fyne's defaults so only
// the colours and sizes we care about change.
func newTelegramTheme() fyne.Theme {
	return telegramTheme{Theme: theme.DefaultTheme()}
}

func (t telegramTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	dark := variant != theme.VariantLight

	switch name {
	case theme.ColorNamePrimary:
		return accent
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case theme.ColorNameHover:
		if dark {
			return darkHover
		}
	case theme.ColorNameHeaderBackground:
		return darkPanel
	case theme.ColorNameInputBackground, theme.ColorNameButton:
		if dark {
			return darkInput
		}
	case theme.ColorNameSelection:
		if dark {
			return color.NRGBA{R: 0x33, G: 0x90, B: 0xEC, A: 0x99}
		}
		return color.NRGBA{R: 0x33, G: 0x90, B: 0xEC, A: 0x55}
	case theme.ColorNameSeparator:
		if dark {
			return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x14}
		}
	case theme.ColorNamePlaceHolder:
		if dark {
			return color.NRGBA{R: 0x8A, G: 0x9A, B: 0xA8, A: 0xff}
		}
	case theme.ColorNameHyperlink:
		return accent
	}

	if dark {
		switch name {
		case theme.ColorNameBackground:
			return darkBg
		case theme.ColorNameForeground, theme.ColorNameMenuBackground:
			return color.NRGBA{R: 0xF2, G: 0xF5, B: 0xF8, A: 0xff}
		case theme.ColorNameInputBorder:
			return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1F}
		case theme.ColorNameError:
			return color.NRGBA{R: 0xE8, G: 0x53, B: 0x4F, A: 0xff}
		case theme.ColorNameScrollBar:
			return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x33}
		}
	}

	return t.Theme.Color(name, variant)
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
