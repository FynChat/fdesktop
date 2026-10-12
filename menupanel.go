package main

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// menuWidth is how wide the side menu slides out.
const menuWidth = 280

// sideMenu is the panel that slides in from the left when the hamburger is
// tapped: who you are, the search box that used to crowd the header, and the
// settings that belong there rather than in a toolbar.
type sideMenu struct {
	panel   *fyne.Container
	scrim   *canvas.Rectangle
	open    bool
	anim    *fyne.Animation
	onClose func()

	// searchSlot holds the channel search box. It is emptied while the
	// Contacts screen is up, where a box labelled "Search channels" would be
	// searching something that is not on screen.
	searchSlot *fyne.Container
	search     *widget.Entry
}

func newSideMenu(client *HTTPClient, prefs *Prefs, side sidebar, onLogout, onClose func(), onContacts func()) *sideMenu {
	name := client.Name
	if name == "" {
		name = "Your account"
	}

	// Account row: avatar, name and where to manage the account.
	account := container.NewBorder(nil, nil,
		container.NewHBox(
			container.NewGridWrap(fyne.NewSquareSize(40), avatar(initial(name), 40)),
			container.NewVBox(
				canvas.NewText(name, theme.ForegroundColor()),
				smallText("Tap to manage your account"),
			),
		), nil, nil)

	// Declared first: the row's own handler updates the note it shows.
	var themeItem *menuItem
	themeItem = newMenuItem(theme.ColorPaletteIcon(), "Theme", themeNote(prefs), func() {
		if prefs == nil {
			return
		}
		prefs.SetDarkTheme(!prefs.DarkTheme())
		prefs.apply()
		themeItem.SetNote(themeNote(prefs))
	})

	// Contacts and the two ways of making a conversation sit above the
	// settings, which is where Telegram keeps them too.
	contacts := newMenuItem(theme.AccountIcon(), "Contacts", "", func() {
		if onContacts != nil {
			onContacts()
		}
	})
	newGroup := newMenuItem(theme.FolderNewIcon(), "New group", "", func() {
		comingSoon("New group")
	})

	privacy := newMenuItem(theme.VisibilityIcon(), "Privacy and security", "", func() {
		comingSoon("Privacy and security")
	})
	settings := newMenuItem(theme.SettingsIcon(), "Settings", "", func() {
		comingSoon("Settings")
	})
	logout := newMenuItem(theme.LogoutIcon(), "Log out", "", func() {
		client.Logout()
		onLogout()
	})

	// The search box lives here now, where it has the full width of the menu
	// instead of crowding the list header.
	searchSlot := container.NewVBox(container.NewPadded(side.search))

	panel := container.NewStack(
		canvas.NewRectangle(panelBackground()),
		container.NewVBox(
			container.NewPadded(account),
			searchSlot,
			contacts, newGroup,
			sectionLabel("Settings"),
			themeItem, privacy, settings,
			layout.NewSpacer(),
			logout,
		),
	)

	return &sideMenu{
		panel:      panel,
		onClose:    onClose,
		searchSlot: searchSlot,
		search:     side.search,
	}
}

// showChannelSearch puts the channel search box in the menu or takes it out.
//
// The box is swapped rather than hidden: Fyne 2.8 crashes the painter when an
// Entry inside a container is hidden and shown again, because it is never laid
// out while hidden and its border is then stroked at a negative radius.
func (m *sideMenu) showChannelSearch(visible bool) {
	if visible {
		m.searchSlot.Objects = []fyne.CanvasObject{container.NewPadded(m.search)}
	} else {
		m.searchSlot.Objects = nil
	}
	m.searchSlot.Refresh()
}

// themeNote spells out what is selected rather than leaving a switch to guess.
func themeNote(prefs *Prefs) string {
	if prefs == nil || prefs.DarkTheme() {
		return "Dark"
	}
	return "Light"
}

// panelBackground is the menu's own surface, slightly apart from the window.
func panelBackground() (c color.Color) {
	if currentIsDark {
		return color.NRGBA{R: 0x17, G: 0x21, B: 0x2B, A: 0xff}
	}
	return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff}
}

// toggle slides the menu in or out.
func (m *sideMenu) toggle(canvasSize fyne.Size) {
	m.open = !m.open
	to := float32(-menuWidth)
	if m.open {
		to = 0
	}

	m.scrim.Show()
	if m.anim != nil {
		m.anim.Stop()
	}

	from := m.panel.Position().X
	m.anim = fyne.NewAnimation(180*time.Millisecond, func(v float32) {
		x := from + (to-from)*v
		m.panel.Move(fyne.NewPos(x, 0))
		m.panel.Resize(fyne.NewSize(menuWidth, canvasSize.Height))
	})
	m.anim.Start()

	if !m.open {
		// Hide after the slide finishes so it does not eat clicks on the
		// window while it is off screen.
		time.AfterFunc(200*time.Millisecond, func() {
			fyne.Do(func() {
				if !m.open {
					m.scrim.Hide()
				}
			})
		})
	}
}

// scrimFor is the dimmed area beside the menu; tapping it closes the menu.
func (m *sideMenu) scrimFor() *canvas.Rectangle {
	if m.scrim == nil {
		m.scrim = canvas.NewRectangle(color.NRGBA{A: 0x99})
		m.scrim.Hide()
	}
	return m.scrim
}

// comingSoon reports a section that exists in the menu but not yet in the app,
// instead of leaving a dead row the user taps for nothing.
func comingSoon(name string) {
	if fyne.CurrentApp() == nil {
		return
	}
	for _, w := range fyne.CurrentApp().Driver().AllWindows() {
		dialog.ShowInformation(name, name+" is not built yet.", w)
		return
	}
}
