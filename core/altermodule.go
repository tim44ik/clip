package core

import (
	"clip/locales"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (a *ClipWindow) editModuleName() {
	a.applyModuleChanges()
	input := widget.NewMultiLineEntry()
	input.SetText(a.selectedModule.Name)
	scroll := container.NewVScroll(input)
	scroll.ScrollToBottom()
	addmoduleDialog := dialog.NewCustomConfirm(
		locales.T(a.modules.CurrentLang, "alter_module_name"),
		locales.T(a.modules.CurrentLang, "ok"),
		locales.T(a.modules.CurrentLang, "cancel"),
		container.NewPadded(
			container.NewBorder(
				canvas.NewText(locales.T(a.modules.CurrentLang, "enter_new_module_name"), color.Black),
				nil, nil, nil, scroll,
			),
		), func(b bool) {
			if b && input.Text != "" {
				a.selectedModule.AlterName(input.Text)
				a.refreshModuleGui()
				a.fullRefresh()
			}
		}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(500, 300))
	addmoduleDialog.Show()
}
