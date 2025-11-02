package core

import (
	"clip/utility"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
)

type SpuWindow struct {
	Window fyne.Window

	selectedModule *Module

	currentScenario *Scenario

	Output map[string]string

	makePDF bool

	cancel context.CancelFunc

	Elms struct {
		title                 *canvas.Text
		vunerabilitiesCheck   *widget.Check
		threadEntry           *widget.Entry
		moduleContentEntry    *widget.Entry
		ModuleOutputEntry     *widget.Entry
		modulesPanel          *fyne.Container
		bottomPanelCheckboxes *fyne.Container
		bottomPanelButtons    *fyne.Container
		activity              *widget.Activity
		menu                  *fyne.MainMenu

		ModuleOutputEntryMutex sync.Mutex
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

func CreateWindow() (a *SpuWindow) {
	a = &SpuWindow{}
	a.Modules.MainModule = &Module{Name: "Главный"}

	a.Profiles.Exists = false
	a.buildWindow(fyne.CurrentApp())
	a.SelectModule(a.Modules.MainModule)

	return
}

func (a *SpuWindow) buildWindow(app fyne.App) {
	a.Window = app.NewWindow("SPU")
	a.Elms.title = canvas.NewText("", color.Black)
	a.Elms.title.TextSize = 16

	a.Elms.moduleContentEntry = widget.NewMultiLineEntry()
	a.Elms.moduleContentEntry.SetPlaceHolder("Команды и переменные для использования во всез модулях")

	a.Elms.ModuleOutputEntry = widget.NewMultiLineEntry()
	a.Elms.ModuleOutputEntry.Disable()
	scroll := container.NewVScroll(a.Elms.ModuleOutputEntry)
	scroll.ScrollToBottom()

	a.Elms.threadEntry = widget.NewEntry()
	a.Elms.threadEntry.SetPlaceHolder("Потоки")
	a.Elms.threadEntry.Validator = utility.NumberValidator(1, 128)

	a.Elms.vunerabilitiesCheck = widget.NewCheck("Сформировать PDF", func(b bool) {
		a.makePDF = b
	})

	a.Elms.modulesPanel = container.NewVBox()

	a.Elms.bottomPanelCheckboxes = container.NewVBox(
		container.NewVBox(a.Elms.vunerabilitiesCheck, a.Elms.threadEntry),
	)

	a.Elms.bottomPanelButtons = container.NewVBox(widget.NewButton("Изменить", func() { a.alter() }),
		widget.NewButton("Удалить", func() { a.delete() }))

	a.Elms.activity = widget.NewActivity()

	a.Window.SetContent(
		container.NewBorder(
			container.NewHBox(
				utility.NewDropButton(theme.FolderOpenIcon(), a.Window.Canvas(), fyne.NewMenu("Профиль",
					fyne.NewMenuItem("Загрузить", func() { a.loadProfile() }),
					fyne.NewMenuItem("Загрузить в новом окне", func() { a.loadProfileInNewWindow() }),
					fyne.NewMenuItem("Сохранить", func() { a.saveProfile() }),
					fyne.NewMenuItem("Сохранить как", func() { a.saveProfileAs() }),
				)),
				utility.NewDropButton(theme.MediaPlayIcon(), a.Window.Canvas(), fyne.NewMenu("Сценарий",
					fyne.NewMenuItem("Начать выполнение сценария", func() { a.beginScenario() }),
					fyne.NewMenuItem("Прервать выполнение сценария", func() { a.interruptScenario() }),
				)),
				utility.NewDropButton(theme.CancelIcon(), a.Window.Canvas(), fyne.NewMenu("Выход",
					fyne.NewMenuItem("Начать выполнение сценария", func() { a.Window.Close() }),
				)),
			),
			nil, nil, nil,
			container.NewBorder(
				nil,
				nil,
				container.NewBorder(
					nil,
					a.Elms.activity,
					nil,
					nil,

					container.NewVScroll(
						container.NewVBox(
							widget.NewButton("Главный", func() { a.selectMainModule() }),
							a.Elms.modulesPanel,
							widget.NewButton("Добавить модуль", func() { a.addModule() }),
						),
					),
				),
				nil,
				container.NewPadded(
					container.NewBorder(
						a.Elms.title,
						container.NewCenter(
							container.NewHBox(a.Elms.bottomPanelCheckboxes,
								a.Elms.bottomPanelButtons),
						),
						nil,
						nil,
						container.NewGridWithRows(2, a.Elms.moduleContentEntry, scroll),
					),
				),
			),
		),
	)

	a.Window.Resize(fyne.NewSize(900, 600))
	a.Window.SetOnClosed(func() { a.Window.Close() })
	a.Elms.activity.Hide()
}

func (a *SpuWindow) SelectModule(m *Module) {
	a.ApplyModuleChanges()
	a.selectedModule = m
	a.Elms.title.Text = fmt.Sprintf("Модуль '%s'", func(s string) string {
		if !strings.Contains(s, "\n") && len(s) < 30 {
			return s
		} else if len(s) > 30 {
			return s[:31] + "..."
		}
		s = strings.ReplaceAll(s, "\n", " ")
		return s[:31] + "..."
	}(m.Name))
	a.Elms.title.Refresh()
	a.Elms.moduleContentEntry.SetText(m.Content)
	a.Elms.ModuleOutputEntry.SetText(m.Output)
	a.Elms.ModuleOutputEntry.Hidden = m == a.Modules.MainModule
	a.Elms.bottomPanelButtons.Hidden = m == a.Modules.MainModule
	a.Elms.ModuleOutputEntry.SetText(m.Output)
	a.Elms.ModuleOutputEntry.CursorRow = strings.Count(m.Output, "\n")
	a.Elms.ModuleOutputEntry.Refresh()
	a.Elms.bottomPanelCheckboxes.Refresh()
}

func (a *SpuWindow) ApplyModuleChanges() {
	if a.selectedModule == nil {
		return
	}
	a.selectedModule.Content = a.Elms.moduleContentEntry.Text
}

func (a *SpuWindow) saveProfile() {
	a.ApplyModuleChanges()
	switch a.Profiles.Exists {
	case true:
		a.makeJson(a.Profiles.Path)
	case false:
		a.saveProfileAs()
	}
}

func (a *SpuWindow) saveProfileAs() {
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

func (a *SpuWindow) makeJson(filename string) error {
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

func (a *SpuWindow) loadProfileInNewWindow() {
	window := CreateWindow()
	if window.loadProfile() {
		window.Window.Show()
	}
}

func (a *SpuWindow) loadProfile() bool {
	f, err := zenity.SelectFile(zenity.Title("Загрузите конфигурацию модулей"),
		zenity.FileFilters{{Name: "JSON Files", Patterns: []string{"*.json"}}})

	if err != nil {
		return false
	}

	err = a.readJson(f)
	if err != nil {
		fmt.Println(err)
		return false
	}
	a.Profiles.Exists = true
	a.Profiles.Path = f
	a.refreshModuleGui()
	return true
}

func (a *SpuWindow) readJson(filepath string) error {
	file, err := os.Open(filepath)

	if err != nil {
		return err
	}

	defer file.Close()
	decoder := json.NewDecoder(file)
	mods := a.Modules
	e := decoder.Decode(&mods)
	if e != nil {
		return e
	}
	a.Modules = mods
	if a.Modules.MainModule == nil {
		a.Modules.MainModule = &Module{Name: "Главный"}
	}
	a.SelectModule(a.Modules.MainModule)
	return nil
}

func (a *SpuWindow) beginScenario() {
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

	var t int
	var err error
	if a.Elms.threadEntry.Text == "" {
		t = 1
	} else {
		t, err = strconv.Atoi(a.Elms.threadEntry.Text)
		if err != nil {
			return
		}
	}
	var f string
	if a.makePDF {
		f, err = zenity.SelectFileSave(zenity.Title("Загрузите конфигурацию модулей"),
			zenity.FileFilters{{Name: "PDF Files", Patterns: []string{"*.pdf"}}}, zenity.ConfirmOverwrite())
		if err != nil {
			f = ""
		}
	}

	if f == "" {
		f = strings.TrimSuffix(a.Profiles.Path, ".json") + time.Now().Format(" 02.01.2006 15-04-05") + ".pdf"
	}
	scenario := NewScenario(a.Modules.MainModule.Content, t, a.makePDF, f, a.Modules.ChildModules)
	a.currentScenario = scenario
	a.Elms.ModuleOutputEntry.Text = ""
	a.Elms.ModuleOutputEntry.Refresh()
	go func() {
		a.Elms.activity.Show()
		a.Elms.activity.Start()
		scenario.BeginScenario(ctx, func(s string, m *Module) {
			fyne.DoAndWait(func() { a.addModuleOutput(m, s) })
		})

		if a.currentScenario == scenario {
			fyne.DoAndWait(func() { a.Elms.activity.Hide() })
			a.currentScenario = nil
			zenity.Info("Выполнение сценария окончено", zenity.Title("Выполнено"))
		}
	}()

}

func (a *SpuWindow) interruptScenario() {
	if a.currentScenario == nil {
		zenity.Info("Сценарий не запущен", zenity.Title("Ошибка"))
		return
	}
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.currentScenario = nil
	a.Elms.activity.Hide()
	zenity.Info("Выполнение сценария прервано", zenity.Title("Прервано"))
}

func (a *SpuWindow) delete() {
	if a.selectedModule == a.Modules.MainModule {
		return
	}
	a.Modules.ChildModules = slices.DeleteFunc(a.Modules.ChildModules, func(m *Module) bool {
		return m == a.selectedModule
	})
	a.selectMainModule()
	a.refreshModuleGui()
}

func (a *SpuWindow) alter() {
	a.ApplyModuleChanges()
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
		a.Elms.title.Text = fmt.Sprintf("Модуль '%s'", m.Name)
		a.Elms.title.Refresh()
		a.refreshModuleGui()
	})
}

func (a *SpuWindow) addModule() {
	a.ApplyModuleChanges()
	ShowModuleCreationDialog(func(m *Module) {
		if m.Name == "" {
			return
		}
		a.Modules.ChildModules = append(a.Modules.ChildModules, m)
		a.Elms.modulesPanel.Add(a.createModuleButton(m))
		a.Elms.modulesPanel.Refresh()
		a.SelectModule(m)
	})
}

func (a *SpuWindow) selectMainModule() {
	a.SelectModule(a.Modules.MainModule)
}

func (a *SpuWindow) createModuleButton(m *Module) fyne.Widget {
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
			return s
		}(m.Name), func() { a.SelectModule(m) })
	}
	return widget.NewButton(m.Name, func() { a.SelectModule(m) })
}

func (a *SpuWindow) refreshModuleGui() {
	a.Elms.modulesPanel.RemoveAll()
	for _, m := range a.Modules.ChildModules {
		a.Elms.modulesPanel.Add(a.createModuleButton(m))
	}
	a.Elms.modulesPanel.Refresh()

	a.Elms.moduleContentEntry.SetText(a.selectedModule.Content)
	a.Elms.ModuleOutputEntry.SetText(a.selectedModule.Output)
}

func (a *SpuWindow) addModuleOutput(module *Module, line string) {
	if line == "Отменено" && module == a.selectedModule {
		module.Output += line
	} else if line != "Отменено" {
		module.Output += line
	}
	if module == a.selectedModule {
		a.Elms.ModuleOutputEntryMutex.Lock()
		defer a.Elms.ModuleOutputEntryMutex.Unlock()
		a.Elms.ModuleOutputEntry.Text += line
		a.Elms.ModuleOutputEntry.CursorRow = strings.Count(module.Output, "\n")
		a.Elms.ModuleOutputEntry.Refresh()
	}
}
