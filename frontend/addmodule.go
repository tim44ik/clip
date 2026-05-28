package frontend

import (
	"clip/locales"
	"clip/models/modules"

	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (a *ClipWindow) addDialog() {
	a.applyModuleChanges()
	input := widget.NewMultiLineEntry()
	scroll := container.NewVScroll(input)
	addmoduleDialog := dialog.NewCustomConfirm(
		locales.T(a.modules.CurrentLang, "add_new_module"),
		locales.T(a.modules.CurrentLang, "ok"),
		locales.T(a.modules.CurrentLang, "cancel"),
		container.NewPadded(
			container.NewBorder(
				canvas.NewText(
					locales.T(a.modules.CurrentLang, "enter_new_module_name"),
					color.Black,
				),
				nil, nil, nil, scroll,
			),
		), func(b bool) {
			if b {
				a.add(input.Text, "")
			} else {
				return
			}
		}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(500, 300))
	addmoduleDialog.Show()
}

func (a *ClipWindow) add(name, content string) {
	if name == "" {
		return
	}
	m := modules.CreateModule(name, content)
	a.modules.ChildModules = append(a.modules.ChildModules, m)
	a.elms.modulesPanel.Add(a.createModuleButton(m))
	a.elms.modulesPanel.Refresh()
	a.applyModuleChanges()
	a.selectModule(m)
}

func (a *ClipWindow) createModuleButton(m *modules.Module) fyne.Widget {
	if len(m.Name) > 18 && !strings.Contains(m.Name, "\n") {
		return widget.NewButton(func(s string) string {
			s = strings.TrimSpace(s)
			runeS := []rune(s)
			if len(runeS) > 18 {
				f := 0
				for i := range runeS {
					if runeS[i] == '+' || runeS[i] == '-' || runeS[i] == '_' || runeS[i] == '=' || runeS[i] == ' ' {
						f = i
					}
					if i%18 == 0 && f != 0 {
						sb := string(runeS[:f])
						sa := string(runeS[f:])
						s = sb + "\n" + sa
						runeS = []rune(s)
						continue
					}
					if i%18 == 0 && i != 0 {
						sb := string(runeS[:i])
						sa := string(runeS[i:])
						s = sb + "\n" + sa
						runeS = []rune(s)
					}
				}
			}
			return string(runeS)
		}(m.Name), func() { a.selectModule(m) })
	}
	return widget.NewButton(strings.TrimSpace(m.Name),
		func() {
			a.applyModuleChanges()
			a.selectModule(m)
		})
}
