package main

import (
	"fmt"
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
	scrim   *scrimWidget
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

	// Enhanced account row with better styling
	userName := canvas.NewText(name, theme.ForegroundColor())
	userName.TextStyle = fyne.TextStyle{Bold: true}
	userName.TextSize = theme.Size(theme.SizeNameSubHeadingText)

	userSub := canvas.NewText("User ID: "+fmt.Sprintf("%d", client.UserID), theme.PlaceHolderColor())
	userSub.TextSize = theme.Size(theme.SizeNameCaptionText)

	avatarWidget := container.NewGridWrap(fyne.NewSquareSize(52), avatar(initial(name), 52))

	accountDetails := container.NewVBox(userName, userSub)
	accountDetails = container.NewPadded(accountDetails)

	accountRow := container.NewHBox(avatarWidget, accountDetails)
	accountRow = container.NewPadded(accountRow)

	// Account row with better styling
	accountBg := canvas.NewRectangle(panelBackground())
	accountBg.CornerRadius = 12

	account := container.NewPadded(container.NewStack(accountBg, container.NewPadded(accountRow)))

	// Declared first: the row's own handler updates the note it shows.
	var themeItem *menuItem
	themeItem = newMenuItem(theme.ColorPaletteIcon(), "Theme", themeNote(prefs), func() {
		if prefs == nil {
			return
		}
		prefs.SetDarkTheme(!prefs.DarkTheme())
		prefs.apply()
		themeItem.SetNote(themeNote(prefs))
		// Close menu after action - we'll do this via scrim tap
	})

	// Contacts and the two ways of making a conversation sit above the
	// settings, which is where Telegram keeps them too.
	contacts := newMenuItem(theme.AccountIcon(), "Contacts", "", func() {
		if onContacts != nil {
			onContacts()
		}
		// Close menu after navigation
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
			account,
			widget.NewSeparator(),
			searchSlot,
			contacts, newGroup,
			sectionLabel("Settings"),
			themeItem, privacy, settings,
			layout.NewSpacer(),
			logout,
		),
	)

	m := &sideMenu{
		panel:      panel,
		onClose:    onClose,
		searchSlot: searchSlot,
		search:     side.search,
	}
	m.buildScrim()
	return m
}

// buildScrim creates the tap-to-close dimmer that sits beside the open menu. It
// is a single tappable rectangle, NOT a button stacked over the window: a Stack
// gives every child the full size, so an invisible button there swallowed every
// tap on the sidebar and the chat underneath.
func (m *sideMenu) buildScrim() {
	m.scrim = newScrimWidget(func() {
		if m.open {
			if app := fyne.CurrentApp(); app != nil {
				if wins := app.Driver().AllWindows(); len(wins) > 0 {
					m.toggle(wins[0].Canvas().Size())
				}
			}
		}
	})
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

// scrimWidget is the dimmed area beside the open menu. It fills the window (the
// overlay Stack sizes it) and closes the menu when tapped anywhere the menu
// panel is not. It is a tappable rectangle rather than a button on top of one:
// a Stack hands every child the full window size, so an invisible button would
// sit over the whole app and swallow every tap on the sidebar and the chat.
type scrimWidget struct {
	widget.BaseWidget
	rect  *canvas.Rectangle
	onTap func()
}

func newScrimWidget(onTap func()) *scrimWidget {
	s := &scrimWidget{
		rect:  canvas.NewRectangle(color.NRGBA{A: 0x99}),
		onTap: onTap,
	}
	s.ExtendBaseWidget(s)
	s.Hide() // hidden until the menu opens
	return s
}

func (s *scrimWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.rect)
}

func (s *scrimWidget) Tapped(*fyne.PointEvent) {
	if s.onTap != nil {
		s.onTap()
	}
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
