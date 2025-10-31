package main

import (
	core "smartpentestutility/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	app.New()
	fyne.CurrentApp().Settings().SetTheme(core.BlackTextTheme{})
	core.CreateWindow().Window.ShowAndRun()
}
