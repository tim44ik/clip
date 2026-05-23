package frontend

import (
	"clip/locales"
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (a *ClipWindow) changeLanguageWindow() {
	a.applyModuleChanges()
	options := func() []string {
		slice := []string{}
		for _, l := range a.langs {
			slice = append(slice, l)
		}
		return slice
	}()
	dropoutMenu := widget.NewSelectEntry(options)
	langwindow := dialog.NewCustomConfirm(
		locales.T(a.modules.CurrentLang, "change_language"),
		locales.T(a.modules.CurrentLang, "apply"),
		locales.T(a.modules.CurrentLang, "cancel"),
		container.NewBorder(
			container.NewVBox(canvas.NewText(
				locales.T(a.modules.CurrentLang, "choose_language"),
				color.Black),
				dropoutMenu),
			nil, nil, nil,
		),
		func(b bool) {
			if slices.Contains(options, dropoutMenu.Text) && b {
				a.modules.CurrentLang = dropoutMenu.Text
				a.fullRefresh()
			}
		},
		a.Window,
	)
	langwindow.Resize(fyne.NewSize(500, 100))
	langwindow.Show()
}
