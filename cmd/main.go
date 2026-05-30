package main

import (
	"clip/frontend"
	"clip/frontend/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	app.NewWithID("123")
	fyne.CurrentApp().Settings().SetTheme(theme.BlackTextTheme{})
	frontend.CreateWindow().Window.ShowAndRun()
}
