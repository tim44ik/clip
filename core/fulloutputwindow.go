package core

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func FullOutput(a *SpuWindow) {
	a.applyModuleChanges()
	input := widget.NewMultiLineEntry()
	input.Disabled()
	input.Text = a.selectedModule.Output
	scroll := container.NewVScroll(input)
	addmoduleDialog := dialog.NewCustomConfirm(
		a.langmap[a.Modules.CurrentLang][33],
		a.langmap[a.Modules.CurrentLang][23],
		a.langmap[a.Modules.CurrentLang][24],
		container.NewPadded(
			container.NewBorder(
				nil, nil, nil, nil, scroll,
			),
		), func(b bool) {}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(800, 600))
	addmoduleDialog.Show()
}
