package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// registerView builds the registration screen: name, username, email and
// password, a Create account button and a link back to login. It reuses the
// authLayout shell from login.go so both screens look like one app.
func registerView(client *HTTPClient, w fyne.Window, onRegisterSuccess func(), onSwitchToLogin func()) fyne.CanvasObject {
	name := widget.NewEntry()
	name.SetPlaceHolder("Name")

	username := widget.NewEntry()
	username.SetPlaceHolder("Username")

	email := widget.NewEntry()
	email.SetPlaceHolder("Email")

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password (8 characters minimum)")

	registerBtn := widget.NewButton("Create account", func() {
		go func() {
			_, err := client.Register(name.Text, username.Text, email.Text, password.Text)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				onRegisterSuccess()
			})
		}()
	})
	password.OnSubmitted = func(string) { registerBtn.OnTapped() }

	fields := []fyne.CanvasObject{name, username, email, password}

	return authLayout("Create your account", "Pick a username — this is how people will see you.",
		fields,
		registerBtn,
		"Already registered?",
		"Sign in",
		onSwitchToLogin)
}
