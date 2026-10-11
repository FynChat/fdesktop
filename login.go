package main

import (
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// loginView builds the login screen: the logo, a heading, email and password
// entries, a Log In button and a link to registration. Input is checked here
// before anything is sent, so the common mistakes never reach the network.
func loginView(client *HTTPClient, w fyne.Window, onLoginSuccess func(), onSwitchToRegister func()) fyne.CanvasObject {
	email := widget.NewEntry()
	email.SetPlaceHolder("Email")
	email.Validator = emailValidator

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password")

	banner := errorBannerWidget()

	loginBtn := asyncButton("Log In", "Signing in…", func() {
		if err := validateLogin(email.Text, password.Text); err != nil {
			banner.Show(err)
			return
		}

		go func() {
			_, err := client.Login(email.Text, password.Text)

			fyne.Do(func() {
				if err != nil {
					banner.Show(err)
					return
				}
				onLoginSuccess()
			})
		}()
	})
	loginBtn.Importance = widget.HighImportance
	password.OnSubmitted = func(string) { loginBtn.OnTapped() }

	return authLayout("Welcome back", "Sign in to continue your conversations.",
		[]fyne.CanvasObject{email, password},
		banner,
		loginBtn,
		"Haven't got an account yet?",
		"Create one",
		onSwitchToRegister)
}

// authLayout is the shared shell of the login and register screens: logo,
// heading, subtitle, the fields, one primary button, an error banner and one
// quiet link. The whole card is centred so the window never looks
// half-finished.
func authLayout(heading, subtitle string, fields []fyne.CanvasObject, banner *errorBanner, primary *widget.Button, linkPrompt, linkLabel string, onLink func()) fyne.CanvasObject {
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
		banner.wrap,
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

// minPasswordLength mirrors the binding rule the backend applies, so a short
// password is caught before a request is made.
const minPasswordLength = 8

// emailValidator marks an entry invalid while its content is not an email.
func emailValidator(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil // empty is handled when the form is submitted
	}
	at := strings.Index(s, "@")
	if at <= 0 || at == len(s)-1 || strings.Contains(s, " ") {
		return errors.New("that does not look like an email address")
	}
	return nil
}

var (
	errNameRequired     = errors.New("tell us your name")
	errUsernameRequired = errors.New("pick a username")
	errEmailRequired    = errors.New("enter your email")
	errPasswordTooShort = fmt.Errorf("use at least %d characters", minPasswordLength)
)

func validateLogin(email, password string) error {
	if strings.TrimSpace(email) == "" {
		return errEmailRequired
	}
	if err := emailValidator(email); err != nil {
		return err
	}
	if password == "" {
		return errors.New("enter your password")
	}
	return nil
}

func validateRegistration(name, username, email, password string) error {
	if strings.TrimSpace(name) == "" {
		return errNameRequired
	}
	if strings.TrimSpace(username) == "" {
		return errUsernameRequired
	}
	if strings.TrimSpace(email) == "" {
		return errEmailRequired
	}
	if err := emailValidator(email); err != nil {
		return err
	}
	if len(password) < minPasswordLength {
		return errPasswordTooShort
	}
	return nil
}
