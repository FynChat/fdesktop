package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// loginView builds the login screen: the logo, a heading, email and password
// entries, a Log In button and a link to registration.
func loginView(client *HTTPClient, w fyne.Window, onLoginSuccess func(), onSwitchToRegister func()) fyne.CanvasObject {
	email := widget.NewEntry()
	email.SetPlaceHolder("Email")

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password")

	loginBtn := widget.NewButton("Log In", func() {
		go func() {
			_, err := client.Login(email.Text, password.Text)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				onLoginSuccess()
			})
		}()
	})
	password.OnSubmitted = func(string) { loginBtn.OnTapped() }

	return authLayout("Welcome back", "Sign in to continue your conversations.",
		[]fyne.CanvasObject{email, password},
		loginBtn,
		"Haven't got an account yet?",
		"Create one",
		onSwitchToRegister)
}

// authLayout is the shared shell of the login and register screens: logo,
// heading, subtitle, the fields, one primary button and one quiet link. The
// whole card is centred so the window never looks half-finished.
func authLayout(heading, subtitle string, fields []fyne.CanvasObject, primary *widget.Button, linkPrompt, linkLabel string, onLink func()) fyne.CanvasObject {
	// High importance paints the button with the accent colour, which is how
	// the primary action stands out from the quiet "Log out" / "Back" buttons.
	primary.Importance = widget.HighImportance

	title := canvas.NewText(heading, theme.ForegroundColor())
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = theme.Size(theme.SizeNameHeadingText)

	sub := canvas.NewText(subtitle, theme.PlaceHolderColor())

	link := newLinkLabel(linkLabel, onLink)

	content := []fyne.CanvasObject{
		container.NewCenter(appLogo(64)),
		container.NewCenter(title),
		container.NewCenter(sub),
		layout.NewSpacer(),
	}
	for _, f := range fields {
		content = append(content, f)
	}
	content = append(content,
		primary,
		container.NewCenter(container.NewVBox(linkPromptLabel(linkPrompt), link)),
	)

	// Center vertically: a NewVBox hugs its content, NewCenter pushes it into
	// the middle of whatever space the window gives it.
	return container.NewCenter(
		container.NewPadded(
			container.NewVBox(content...),
		),
	)
}

// linkPromptLabel is the small grey "no account yet?" text above a hyperlink.
func linkPromptLabel(text string) fyne.CanvasObject {
	l := canvas.NewText(text, theme.PlaceHolderColor())
	l.TextSize = theme.Size(theme.SizeNameCaptionText)
	return l
}
