package main

import (
	"clip/core"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "-run-isolated" {
		runIso(os.Args)
		return
	}
	app.NewWithID("123")
	fyne.CurrentApp().Settings().SetTheme(core.BlackTextTheme{})
	core.CreateWindow().Window.ShowAndRun()
}
