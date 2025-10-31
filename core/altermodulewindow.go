package core

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowModuleAlteringDialog(oldModule *Module, callback func(*Module)) {
	window := fyne.CurrentApp().NewWindow("Изменение модуля")

	title := canvas.NewText("Название:", color.Black)
	title.TextSize = 16

	input := widget.NewEntry()
	input.SetText(oldModule.Name)
	createButton := widget.NewButton("Сохранить модуль", func() {
		newModule := &Module{
			Name:    input.Text,
			Content: oldModule.Content,
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
