package core

import (
	"clip/modules"

	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func AddModule(a *ClipWindow) {
	a.applyModuleChanges()
	input := widget.NewMultiLineEntry()
	scroll := container.NewVScroll(input)
	addmoduleDialog := dialog.NewCustomConfirm(
		a.langmap[a.Modules.CurrentLang][26],
		a.langmap[a.Modules.CurrentLang][23],
		a.langmap[a.Modules.CurrentLang][24],
		container.NewPadded(
			container.NewBorder(
				canvas.NewText(a.langmap[a.Modules.CurrentLang][25], color.Black),
				nil, nil, nil, scroll,
			),
		), func(b bool) {
			addDialog(a, input, b)
		}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(500, 300))
	addmoduleDialog.Show()
}

func addDialog(a *ClipWindow, input *widget.Entry, b bool) {
	if b {
		if input.Text == "" {
			return
		}
		m := &modules.Module{
			Name:    input.Text,
			Content: "",
			Output:  "",
		}
		a.Modules.ChildModules = append(a.Modules.ChildModules, m)
		a.elms.modulesPanel.Add(CreateModuleButton(a, m))
		a.elms.modulesPanel.Refresh()
		a.applyModuleChanges()
		a.selectModule(m)
	} else {
		return
	}
}

func CreateModuleButton(a *ClipWindow, m *modules.Module) fyne.Widget {
	if len(m.Name) > 18 && !strings.Contains(m.Name, "\n") {
		return widget.NewButton(func(s string) string {
			if len(s) > 18 {
				f := 0
				for i := range s {
					if s[i] == '+' || s[i] == '-' || s[i] == '_' || s[i] == '=' || s[i] == ' ' {
						f = i
					}
					if i%18 == 0 && f != 0 {
						s = s[:f] + "\n" + s[f:]
						continue
					}
					if i%18 == 0 && i != 0 {
						s = s[:i] + "\n" + s[i:]
					}
				}
			}
			return strings.TrimSpace(s)
		}(m.Name), func() { a.selectModule(m) })
	}
	return widget.NewButton(strings.TrimSpace(m.Name),
		func() {
			a.applyModuleChanges()
			a.selectModule(m)
		})
}
