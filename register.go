package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// registerView builds the registration screen: name, username, email and
// password entries, a Register button, and a button to switch to login.
func registerView(client *HTTPClient, w fyne.Window, onRegisterSuccess func(), onSwitchToLogin func()) fyne.CanvasObject {
	name := widget.NewEntry()
	name.SetPlaceHolder("Name")

	username := widget.NewEntry()
	username.SetPlaceHolder("Username")

	email := widget.NewEntry()
	email.SetPlaceHolder("Email")

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password")

	registerBtn := widget.NewButton("Register", func() {
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

	switchBtn := widget.NewButton("Have an account? Login", onSwitchToLogin)

	return container.NewVBox(
		widget.NewLabel("Register"),
		name,
		username,
		email,
		password,
		registerBtn,
		switchBtn,
	)
}
