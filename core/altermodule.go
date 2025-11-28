package core

import (
	"fmt"
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func Alter(a *SpuWindow) {
	a.applyModuleChanges()
	input := widget.NewMultiLineEntry()
	input.SetText(a.selectedModule.Name)
	scroll := container.NewVScroll(input)
	scroll.ScrollToBottom()
	addmoduleDialog := dialog.NewCustomConfirm(
		a.langmap[a.Modules.CurrentLang][22],
		a.langmap[a.Modules.CurrentLang][23],
		a.langmap[a.Modules.CurrentLang][24],
		container.NewPadded(
			container.NewBorder(
				canvas.NewText(a.langmap[a.Modules.CurrentLang][25], color.Black),
				nil, nil, nil, scroll,
			),
		), func(b bool) {
			alterdialog(a, input, b)
		}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(500, 300))
	addmoduleDialog.Show()
}

func alterdialog(a *SpuWindow, input *widget.Entry, b bool) {
	if b {
		if input.Text == "" {
			return
		}
		m := &Module{
			Name:    input.Text,
			Content: a.selectedModule.Content,
			Output:  a.selectedModule.Output,
		}
		a.Modules.ChildModules[slices.Index(a.Modules.ChildModules, a.selectedModule)] = m
		a.selectedModule = m
		a.elms.title.Text = fmt.Sprintf("%s '%s'", a.langmap[a.Modules.CurrentLang][15], m.Name)
		a.elms.title.Refresh()
		a.refreshModuleGui()
	} else {
		return
	}
}
