package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// loginView builds the login screen: email/password entries, a Login
// button, and a button to switch to registration.
func loginView(client *HTTPClient, w fyne.Window, onLoginSuccess func(), onSwitchToRegister func()) fyne.CanvasObject {
	email := widget.NewEntry()
	email.SetPlaceHolder("Email")

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password")

	loginBtn := widget.NewButton("Login", func() {
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

	switchBtn := widget.NewButton("Need an account? Register", onSwitchToRegister)

	return container.NewVBox(
		widget.NewLabel("Login"),
		email,
		password,
		loginBtn,
		switchBtn,
	)
}
