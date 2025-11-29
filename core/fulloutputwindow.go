package core

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func FullOutput(a *SpuWindow) {
	a.applyModuleChanges()
	input := widget.NewMultiLineEntry()
	input.Disabled()
	input.Text = strings.Join(a.selectedModule.Output, "")
	scroll := container.NewVScroll(input)
	addmoduleDialog := dialog.NewCustomConfirm(
		"",
		a.langmap[a.Modules.CurrentLang][23],
		a.langmap[a.Modules.CurrentLang][24],
		container.NewPadded(
			container.NewBorder(
				canvas.NewText(a.langmap[a.Modules.CurrentLang][25], color.Black),
				nil, nil, nil, scroll,
			),
		), func(b bool) {}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(800, 600))
	addmoduleDialog.Show()
}
