package core

import (
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ChangeLanguageWindow(a *ClipWindow) {
	a.applyModuleChanges()
	options := func(langmap map[string][]string) []string {
		slice := []string{}
		for key := range langmap {
			slice = append(slice, key)
		}
		return slice
	}(a.langmap)
	dropoutMenu := widget.NewSelectEntry(options)
	langwindow := dialog.NewCustomConfirm(
		a.langmap[a.Modules.CurrentLang][29],
		a.langmap[a.Modules.CurrentLang][30],
		a.langmap[a.Modules.CurrentLang][24],
		container.NewBorder(
			container.NewVBox(canvas.NewText(
				a.langmap[a.Modules.CurrentLang][31],
				color.Black),
				dropoutMenu),
			nil, nil, nil,
		),
		func(b bool) {
			if slices.Contains(options, dropoutMenu.Text) {
				a.Modules.CurrentLang = dropoutMenu.Text
				a.fullrefresh()
			}
		},
		a.Window,
	)
	langwindow.Resize(fyne.NewSize(500, 100))
	langwindow.Show()
}
