package main

import (
	"clip/core"
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	args := os.Args
	if len(args) > 2 && args[1] == "-run-isolated" {
		cmd := exec.Command(args[2], args[3:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			println(err.Error())
			return
		}
		return
	}

	app.NewWithID("123")
	fyne.CurrentApp().Settings().SetTheme(core.BlackTextTheme{})
	core.CreateWindow().Window.ShowAndRun()
}
