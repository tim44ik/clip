package core

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"smartpentestutility/utility"
	"strconv"

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

	currentScenario *Scenario

	Output map[string]string

	makePDF bool

	cancel context.CancelFunc

	elms struct {
		title                 *canvas.Text
		vunerabilitiesCheck   *widget.Check
		threadEntry           *widget.Entry
		moduleContentEntry    *widget.Entry
		modulesPanel          *fyne.Container
		bottomPanelCheckboxes *fyne.Container
		bottomPanelButtons    *fyne.Container
		activity              *widget.Activity
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
	// if a.Window == nil {
	a.Window = a.NewWindow("SPU")
	// } else {
	// 	a.Window = a.NewWindow(path.Base(a.Profiles.Path))
	// }
	a.elms.title = canvas.NewText("", color.Black)
	a.elms.title.TextSize = 16

	a.elms.moduleContentEntry = widget.NewMultiLineEntry()
	a.elms.moduleContentEntry.SetPlaceHolder("Команды и переменные для использования во всез модулях")
	a.elms.moduleContentEntry.OnChanged = utility.EntryAutoexpand(a.elms.moduleContentEntry, 3, 20)

	a.elms.threadEntry = widget.NewEntry()
	a.elms.threadEntry.SetPlaceHolder("Потоки")
	a.elms.threadEntry.Validator = utility.NumberValidator(1, 128)

	a.elms.vunerabilitiesCheck = widget.NewCheck("Сформировать PDF", func(b bool) {
		a.makePDF = b
	})

	a.elms.modulesPanel = container.NewVBox()

	a.elms.bottomPanelCheckboxes = container.NewVBox(
		container.NewVBox(a.elms.vunerabilitiesCheck, a.elms.threadEntry),
	)

	a.elms.bottomPanelButtons = container.NewVBox(widget.NewButton("Изменить", func() { a.alter() }),
		widget.NewButton("Удалить", func() { a.delete() }))

	a.elms.activity = widget.NewActivity()

	a.Window.SetContent(
		container.NewStack(
			canvas.NewRectangle(color.White),
			container.NewBorder(
				nil,
				nil,
				container.NewBorder(
					nil,
					a.elms.activity,
					nil,
					nil,

					container.NewVScroll(
						container.NewVBox(
							widget.NewButton("Главный", func() { a.selectMainModule() }),
							a.elms.modulesPanel,
							widget.NewButton("Добавить модуль", func() { a.addModule() }),
						),
					),
				),
				nil,
				container.NewPadded(
					container.NewBorder(
						container.NewVBox(a.elms.title, a.elms.moduleContentEntry),
						container.NewCenter(
							container.NewHBox(a.elms.bottomPanelCheckboxes,
								a.elms.bottomPanelButtons),
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
				fyne.NewMenuItem("Загрузить в новом окне", func() { a.loadProfileInNewWindow() }),
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

	a.elms.activity.Hide()
}

func (a *SpuApp) SelectModule(m *Module) {
	a.ApplyModuleChanges()
	a.selectedModule = m
	a.elms.title.Text = fmt.Sprintf("Модуль '%s'", m.Name)
	a.elms.title.Refresh()
	a.elms.moduleContentEntry.SetText(m.Content)
	a.elms.bottomPanelButtons.Hidden = m == a.Modules.MainModule
	a.elms.bottomPanelCheckboxes.Refresh()
}

func (a *SpuApp) ApplyModuleChanges() {
	if a.selectedModule == nil {
		return
	}
	a.selectedModule.Content = a.elms.moduleContentEntry.Text
}

func (a *SpuApp) saveProfile() {
	a.ApplyModuleChanges()
	switch a.Profiles.Exists {
	case true:
		a.makeJson(a.Profiles.Path)
	case false:
		a.saveProfileAs()
	}
}

func (a *SpuApp) saveProfileAs() {
	a.ApplyModuleChanges()
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

func (a *SpuApp) loadProfileInNewWindow() {
	f, err := zenity.SelectFile(zenity.Title("Загрузите конфигурацию модулей"),
		zenity.FileFilters{{Name: "JSON Files", Patterns: []string{"*.json"}}})

	if err != nil {
		return
	}

	a.buildWindow()

	err = a.readJson(f)
	if err != nil {
		fmt.Println(err)
		return
	}

	a.Profiles.Exists = true
	a.Profiles.Path = f
	a.refreshModuleGui()
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
	a.ApplyModuleChanges()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	if len(a.Modules.ChildModules) == 0 {
		return
	}
	if a.currentScenario != nil {
		zenity.Info("Сценарий уже запущен", zenity.Title("Ошибка"))
		return
	}

	nameCommandDict := make(map[string]string)
	for _, c := range a.Modules.ChildModules {
		nameCommandDict[c.Name] = c.Content
	}
	var t int
	var err error
	if a.elms.threadEntry.Text == "" {
		t = 1
	} else {
		t, err = strconv.Atoi(a.elms.threadEntry.Text)
		if err != nil {
			return
		}
	}
	scenario := NewScenario(a.Modules.MainModule.Content, nameCommandDict, t, a.Profiles.Path, a.makePDF)
	a.currentScenario = scenario
	go func() {
		a.elms.activity.Show()
		a.elms.activity.Start()

		scenario.BeginScenario(ctx)

		if a.currentScenario == scenario {
			a.elms.activity.Hide()
			a.currentScenario = nil
			zenity.Info("Выполнение сценария окончено", zenity.Title("Выполнено"))
		}
	}()
}

func (a *SpuApp) interruptScenario() {
	if a.currentScenario == nil {
		zenity.Info("Сценарий не запущен", zenity.Title("Ошибка"))
		return
	}
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.currentScenario = nil
	a.elms.activity.Hide()
	zenity.Info("Выполнение сценария прервано", zenity.Title("Прервано"))
}

func (a *SpuApp) delete() {
	if a.selectedModule == a.Modules.MainModule {
		return
	}
	a.Modules.ChildModules = slices.DeleteFunc(a.Modules.ChildModules, func(m *Module) bool {
		return m == a.selectedModule
	})
	a.selectMainModule()
	a.refreshModuleGui()
}

func (a *SpuApp) alter() {
	ShowModuleAlteringDialog(a.selectedModule, func(m *Module) {
		if m == a.Modules.MainModule {
			return
		}
		if m.Name == "" {
			return
		}
		if m.Name == a.selectedModule.Name {
			return
		}
		a.Modules.ChildModules[slices.Index(a.Modules.ChildModules, a.selectedModule)] = m
		a.selectedModule = m
		a.elms.title.Text = fmt.Sprintf("Модуль '%s'", m.Name)
		a.elms.title.Refresh()
		a.refreshModuleGui()
	})
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
