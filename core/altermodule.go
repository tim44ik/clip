package core

import (
	"clip/modules"

	"image/color"
	"slices"
	"strings"

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
		a.langmap[a.modules.CurrentLang][22],
		a.langmap[a.modules.CurrentLang][23],
		a.langmap[a.modules.CurrentLang][24],
		container.NewPadded(
			container.NewBorder(
				canvas.NewText(a.langmap[a.modules.CurrentLang][25], color.Black),
				nil, nil, nil, scroll,
			),
		), func(b bool) {
			editdialog(a, input, b)
		}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(500, 300))
	addmoduleDialog.Show()
}

func editdialog(a *ClipWindow, input *widget.Entry, b bool) {
	if b {
		if input.Text == "" {
			return
		}
		m := &modules.Module{
			Name:    strings.TrimSpace(input.Text),
			Content: a.selectedModule.Content,
			Output:  a.selectedModule.Output,
		}
		a.modules.ChildModules[slices.Index(a.modules.ChildModules, a.selectedModule)] = m
		a.selectModule(m)
		a.refreshModuleGui()
	} else {
		return
	}
}
