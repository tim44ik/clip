package core

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowModuleCreationDialog(callback func(*Module)) {
	window := SpuAppInstance.NewWindow("Добавление модуля")

	title := canvas.NewText("Название:", color.Black)
	title.TextSize = 16

	input := widget.NewEntry()

	createButton := widget.NewButton("Добавить модуль", func() {
		newModule := &Module{
			Name:    input.Text,
			Content: "",
		}
		callback(newModule)
		window.Close()
	})

	centralContent := container.NewBorder(container.NewVBox(title, input),
		createButton,
		nil, nil)
	centralPadded := container.NewPadded(centralContent)
	background := canvas.NewRectangle(color.White)
	content := container.NewStack(background, centralPadded)

	window.SetContent(content)
	window.Resize(fyne.NewSize(500, 300))
	window.SetFixedSize(true)
	window.Show()

	window.Canvas().Focus(input)
}
