package frontend

import (
	"clip/locales"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (a *ClipWindow) fullOutput() {
	a.applyModuleChanges()
	output := widget.NewMultiLineEntry()
	output.Disabled()
	output.Text = a.selectedModule.Output
	scroll := container.NewVScroll(output)
	addmoduleDialog := dialog.NewCustomConfirm(
		locales.T(a.modules.CurrentLang, "full_output"),
		locales.T(a.modules.CurrentLang, "ok"),
		locales.T(a.modules.CurrentLang, "cancel"),
		container.NewPadded(
			container.NewBorder(
				nil, nil, nil, nil, scroll,
			),
		), func(b bool) {}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(800, 600))
	addmoduleDialog.Show()
}
