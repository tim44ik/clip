package core

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowModuleCreationDialog(callback func(*Module)) {
	window := fyne.CurrentApp().NewWindow("Добавление модуля")

	title := canvas.NewText("Название:", color.Black)
	title.TextSize = 16

	input := widget.NewMultiLineEntry()

	createButton := widget.NewButton("Добавить модуль", func() {
		newModule := &Module{
			Name:    strings.Trim(input.Text, " \n\t"),
			Content: "",
		}
		callback(newModule)
		window.Close()
	})
	scroll := container.NewVScroll(input)
	scroll.ScrollToBottom()
	centralContent := container.NewBorder(nil,
		createButton,
		nil, nil, container.NewPadded(
			container.NewBorder(
				title,
				nil, nil, nil, scroll,
			),
		),
	)
	background := canvas.NewRectangle(color.White)
	content := container.NewStack(background, centralContent)

	window.SetContent(content)
	window.Resize(fyne.NewSize(500, 300))
	window.Show()

	window.Canvas().Focus(input)
}
