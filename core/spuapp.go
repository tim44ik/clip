package core

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
)

type SpuApp struct {
	fyne.App
	Window fyne.Window

	selectedModule *Module

	elms struct {
		title               *canvas.Text
		vunerabilitiesCheck *widget.Check
		threadEntry         *widget.Entry
		moduleContentEntry  *widget.Entry
		modulesPanel        *fyne.Container
		bottomPanel         *fyne.Container
	}

	Modules struct {
		MainModule   *Module   `json:"MainModule"`
		ChildModules []*Module `json:"ChildModules"`
	}

	Profiles struct {
		Exists bool
		Path   string
	}
}

var SpuAppInstance *SpuApp

func CreateApp() (a *SpuApp) {
	a = &SpuApp{App: app.New()}
	a.Modules.MainModule = &Module{Name: "Главный"}

	a.Profiles.Exists = false
	a.App.Settings().SetTheme(&blackTextTheme{})
	a.buildWindow()
	a.SelectModule(a.Modules.MainModule)

	return
}

func (a *SpuApp) buildWindow() {
	a.Window = a.NewWindow("SPU")

	a.elms.title = canvas.NewText("", color.Black)
	a.elms.title.TextSize = 16

	a.elms.moduleContentEntry = widget.NewMultiLineEntry()
	a.elms.moduleContentEntry.SetPlaceHolder("Строка для автовставки в модули")
	a.elms.moduleContentEntry.OnChanged = entryAutoexpand(a.elms.moduleContentEntry, 3, 20)

	a.elms.threadEntry = widget.NewEntry()
	a.elms.threadEntry.SetPlaceHolder("Потоки")
	a.elms.threadEntry.Validator = numberValidator(1, 128)

	a.elms.vunerabilitiesCheck = widget.NewCheck("Искать уязвимости сервисов", func(bool) {})

	a.elms.modulesPanel = container.NewVBox()

	a.elms.bottomPanel = container.NewVBox(
		container.NewVBox(a.elms.vunerabilitiesCheck, a.elms.threadEntry),
		widget.NewButton("Отправить", func() { a.submit() }),
	)

	a.Window.SetContent(
		container.NewStack(
			canvas.NewRectangle(color.White),
			container.NewBorder(
				nil,
				nil,
				container.NewVScroll(
					container.NewVBox(
						widget.NewButton("Главный", func() { a.selectMainModule() }),
						a.elms.modulesPanel,
						widget.NewButton("Добавить модуль", func() { a.addModule() }),
					),
				),
				nil,
				container.NewPadded(
					container.NewBorder(
						container.NewVBox(a.elms.title, a.elms.moduleContentEntry),
						container.NewCenter(
							a.elms.bottomPanel,
						),
						nil,
						nil,
					),
				),
			),
		),
	)

	a.Window.SetMainMenu(
		fyne.NewMainMenu(
			fyne.NewMenu("Профиль",
				fyne.NewMenuItem("Загрузить", func() { a.loadProfile() }),
				fyne.NewMenuItem("Сохранить", func() { a.saveProfile() }),
				fyne.NewMenuItem("Сохранить как", func() { a.saveProfileAs() }),
			), fyne.NewMenu("Сценарий",
				fyne.NewMenuItem("Начать сценарий", func() { a.beginScenario() }),
				fyne.NewMenuItem("Прервать сценарий", func() { a.interruptScenario() }),
			), fyne.NewMenu("Программа",
				fyne.NewMenuItem("Выход", func() { a.App.Quit() }),
			),
		),
	)

	a.Window.Resize(fyne.NewSize(900, 600))
	a.Window.SetFixedSize(true)
}

func (a *SpuApp) SelectModule(m *Module) {
	a.ApplyModuleChanges()
	a.selectedModule = m
	a.elms.title.Text = fmt.Sprintf("Модуль '%s'", m.Name)
	a.elms.title.Refresh()
	a.elms.moduleContentEntry.SetText(m.Content)
	a.elms.bottomPanel.Hidden = m != a.Modules.MainModule
	a.elms.bottomPanel.Refresh()
}

func (a *SpuApp) ApplyModuleChanges() {
	if a.selectedModule == nil {
		return
	}
	a.selectedModule.Content = a.elms.moduleContentEntry.Text
}

func (a *SpuApp) saveProfile() {
	switch a.Profiles.Exists {
	case true:
		a.makeJson(a.Profiles.Path)
	case false:
		a.saveProfileAs()
	}
}

func (a *SpuApp) saveProfileAs() {
	f, err := zenity.SelectFileSave(zenity.Title("Сохраните конфигурацию модулей"),
		zenity.FileFilters{{Name: "JSON Files", Patterns: []string{"*.json"}}},
		zenity.ConfirmOverwrite())

	if err != nil {
		return
	}

	err = a.makeJson(f)
	if err != nil {
		return
	}
	a.Profiles.Exists = true
	a.Profiles.Path = f
}

func (a *SpuApp) makeJson(filename string) error {
	if filepath.Ext(filename) != ".json" {
		filename += ".json"
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	return encoder.Encode(a.Modules)
}

func (a *SpuApp) loadProfile() {
	f, err := zenity.SelectFile(zenity.Title("Загрузите конфигурацию модулей"),
		zenity.FileFilters{{Name: "JSON Files", Patterns: []string{"*.json"}}})

	if err != nil {
		return
	}

	err = a.readJson(f)
	if err != nil {
		fmt.Println(err)
		return
	}
	a.Profiles.Exists = true
	a.Profiles.Path = f
	a.refreshModuleGui()
	a.reporter()
}

func (a *SpuApp) readJson(filepath string) error {
	file, err := os.Open(filepath)

	if err != nil {
		return err
	}

	defer file.Close()
	decoder := json.NewDecoder(file)
	return decoder.Decode(&a.Modules)
}

func (a *SpuApp) beginScenario() {
	a.reporter()
}

func (a *SpuApp) interruptScenario() {

}

func (a *SpuApp) submit() {

}

func (a *SpuApp) addModule() {
	ShowModuleCreationDialog(func(m *Module) {
		if m.Name == "" {
			return
		}
		a.Modules.ChildModules = append(a.Modules.ChildModules, m)
		a.elms.modulesPanel.Add(a.createModuleButton(m))
		a.elms.modulesPanel.Refresh()
	})
}

func (a *SpuApp) selectMainModule() {
	a.SelectModule(a.Modules.MainModule)
}

func (a *SpuApp) createModuleButton(m *Module) fyne.Widget {
	return widget.NewButton(m.Name, func() { a.SelectModule(m) })
}

func (a *SpuApp) refreshModuleGui() {
	a.elms.modulesPanel.RemoveAll()
	for _, m := range a.Modules.ChildModules {
		a.elms.modulesPanel.Add(a.createModuleButton(m))
	}
	a.elms.modulesPanel.Refresh()

	a.elms.moduleContentEntry.SetText(a.selectedModule.Content)

}

func (a *SpuApp) reporter() {
	if len(a.Modules.ChildModules) == 0 {
		return
	}
	var contentArray []string
	fmt.Println(a.Modules.ChildModules)
	for _, c := range a.Modules.ChildModules {
		contentArray = append(contentArray, c.Content)
	}

	var nameArray []string
	for _, c := range a.Modules.ChildModules {
		nameArray = append(nameArray, c.Name)
	}

	// var nameCommandDict map[string]string
	// for index := range nameArray {
	// 	nameCommandDict[nameArray[index]] = contentArray[index]
	// }
	// fmt.Println(nameCommandDict)
}
