package core

import (
	"clip/utility"
	"context"
	_ "embed"
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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/phpdave11/gofpdf"
)

type SpuWindow struct {
	Window fyne.Window

	selectedModule *Module

	currentScenario *Scenario

	langmap map[string][]string

	makePDF struct {
		do      bool
		pdfPath string
	}

	cancel context.CancelFunc

	Elms struct {
		title                  *canvas.Text
		threadEntry            *widget.Entry
		moduleContentEntry     *widget.Entry
		ModuleOutputEntry      *widget.Entry
		modulesPanel           *fyne.Container
		bottomPanelCheckboxes  *fyne.Container
		bottomPanelButtons     *fyne.Container
		topPanel               *fyne.Container
		activity               *widget.Activity
		mainButton             *fyne.Container
		addButton              *fyne.Container
		ModuleOutputEntryMutex sync.Mutex
	}

	Modules struct {
		CurrentLang  string    `json:"CurrentLang"`
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
	a.langmap = make(map[string][]string)
	a.langmap["English"] =
		[]string{"Main",
			"Threads Number",
			"Make PDF report       ",
			"Edit", "Delete",
			"Load", "Load in new window",
			"Save", "Save as",
			"Begin scenario", "Break scenario",
			"Break scenario and make PDF",
			"Change language", "Exit",
			"Add module", "Module", "Scenario is already started",
			"Completed", "Scenario execution completed",
			"Scenario was not started",
			"Interrupted", "Scenario execution was interrupted",
			"Alter module name", "OK", "Cancel",
			"Enter new module name",
			"Add new module", "Cancelled",
			"Error occured while making PDF",
			"Change language", "Apply",
			"Choose language"}

	a.langmap["Русский"] =
		[]string{"Главная",
			"Количество потоков",
			"Сформировать PDF отчёт",
			"Изменить", "Удалить",
			"Загрузить",
			"Загрузить в новом окне",
			"Сохранить", "Сохранить как",
			"Начать сценарий",
			"Прервать сценарий",
			"Прервать сценарий и сформировать отчёт в PDF",
			"Изменить язык", "Выйти",
			"Добавить модуль", "Модуль",
			"Сценарий уже запущен",
			"Выполнено",
			"Выполнение сценария окончено",
			"Сценарий не запущен",
			"Прервано",
			"Выполнение сценария было прервано",
			"Изменить название модуля", "OK",
			"Отмена", "Введите название",
			"Добавить новый модуль", "Отменено",
			"Ошибка при создании PDF",
			"Изменить язык", "Применить",
			"Выберите язык"}

	if a.Modules.CurrentLang == "" {
		a.Modules.CurrentLang = "English"
	}

	a.Modules.MainModule = &Module{Name: a.langmap[a.Modules.CurrentLang][0]}

	a.Profiles.Exists = false
	a.buildWindow(fyne.CurrentApp())
	a.SelectModule(a.Modules.MainModule)

	return
}

func (a *SpuWindow) buildWindow(app fyne.App) {
	a.Window = app.NewWindow("clip")
	a.Elms.title = canvas.NewText(a.langmap[a.Modules.CurrentLang][0], color.Black)
	a.Elms.title.TextSize = 16

	a.Elms.moduleContentEntry = widget.NewMultiLineEntry()
	a.Elms.moduleContentEntry.SetPlaceHolder("")

	a.Elms.ModuleOutputEntry = widget.NewMultiLineEntry()
	a.Elms.ModuleOutputEntry.Disable()

	a.Elms.threadEntry = widget.NewEntry()
	a.Elms.threadEntry.SetPlaceHolder(a.langmap[a.Modules.CurrentLang][1])
	a.Elms.threadEntry.Validator = utility.NumberValidator(1, 128)

	a.Elms.modulesPanel = container.NewVBox()

	a.Elms.bottomPanelCheckboxes = container.NewVBox(
		container.NewVBox(widget.NewCheck(a.langmap[a.Modules.CurrentLang][2], func(b bool) {
			a.makePDF.do = b
		}), a.Elms.threadEntry),
	)

	a.Elms.bottomPanelButtons = container.NewVBox(
		widget.NewButton(a.langmap[a.Modules.CurrentLang][3], func() { a.alter() }),
		widget.NewButton(a.langmap[a.Modules.CurrentLang][4], func() { a.delete() }))

	a.Elms.activity = widget.NewActivity()
	a.Elms.mainButton = container.NewVBox(widget.NewButton(a.langmap[a.Modules.CurrentLang][0], func() {
		a.ApplyModuleChanges()
		a.selectMainModule()
	}))
	a.Elms.addButton = container.NewVBox(widget.NewButton(a.langmap[a.Modules.CurrentLang][14], func() { a.addModule() }))

	a.Elms.topPanel = container.NewHBox(
		utility.NewDropButton(theme.FolderOpenIcon(), a.Window.Canvas(), fyne.NewMenu("Profiles",
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][5], func() { a.loadProfile() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][6], func() { a.loadProfileInNewWindow() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][7], func() { a.saveProfile() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][8], func() { a.saveProfileAs() }),
		)),
		utility.NewDropButton(theme.MediaPlayIcon(), a.Window.Canvas(), fyne.NewMenu("Scenario",
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][9], func() { a.beginScenario() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][10], func() { a.interruptScenario() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][11], func() {
				a.interruptScenario()
				a.PDFcreationWindow()
			}),
		)),
		utility.NewDropButton(theme.SettingsIcon(), a.Window.Canvas(), fyne.NewMenu("Change Language",
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][12], func() { a.changeLanguageWindow() }))),
		utility.NewDropButton(theme.CancelIcon(), a.Window.Canvas(), fyne.NewMenu("Quit",
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][13], func() { a.Window.Close() }),
		)),
	)

	a.Window.SetContent(
		container.NewBorder(
			a.Elms.topPanel,
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
							a.Elms.mainButton,
							a.Elms.modulesPanel,
							a.Elms.addButton,
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
						container.NewGridWithRows(2, a.Elms.moduleContentEntry, a.Elms.ModuleOutputEntry),
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

	a.selectedModule = m
	a.Elms.title.Text = fmt.Sprintf("%s '%s'", a.langmap[a.Modules.CurrentLang][15], func(s string) string {
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
	a.Elms.ModuleOutputEntry.CursorRow = strings.LastIndexAny(m.Output, "\n")
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

	filesavedialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				return
			}
			if writer == nil {
				return
			}
			path := writer.URI().Path()
			writer.Close()

			err = a.makeJson(path)
			if err != nil {
				dialog.ShowError(err, a.Window)
				return
			}

			a.Profiles.Exists = true
			if filepath.Ext(path) != ".json" {
				defer os.Remove(path)
			}

		}, a.Window)
	filesavedialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	filesavedialog.Resize(fyne.NewSize(900, 500))
	filesavedialog.Show()
}

func (a *SpuWindow) makeJson(filename string) error {
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	filename += ".json"
	var outputarray []string
	for _, m := range a.Modules.ChildModules {
		outputarray = append(outputarray, m.Output)
		m.Output = ""
	}
	defer a.restoreOutput(outputarray)
	a.Profiles.Path = filename
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	return encoder.Encode(a.Modules)
}

func (a *SpuWindow) restoreOutput(outputarray []string) {
	for i, m := range a.Modules.ChildModules {
		m.Output = outputarray[i]
	}
}

func (a *SpuWindow) loadProfileInNewWindow() {

	fileOpenDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, a.Window)
				return
			}
			if reader == nil {
				return
			}

			newWindow := CreateWindow()
			filename := reader.URI().Path()
			reader.Close()

			err = newWindow.readJson(filename)
			if err != nil {
				dialog.ShowError(err, a.Window)
				return
			}

			newWindow.Profiles.Exists = true
			newWindow.Profiles.Path = filename
			newWindow.fullrefresh()

			newWindow.Window.Show()
		},
		a.Window,
	)

	fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	fileOpenDialog.Resize(fyne.NewSize(900, 500))
	fileOpenDialog.Show()
}

func (a *SpuWindow) loadProfile() {
	fileOpenDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, a.Window)
				return
			}
			if reader == nil {
				return
			}

			filename := reader.URI().Path()
			reader.Close()

			err = a.readJson(filename)
			if err != nil {
				dialog.ShowError(err, a.Window)
				return
			}

			a.Profiles.Exists = true
			a.Profiles.Path = filename
			a.refreshModuleGui()
		},
		a.Window,
	)

	fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	fileOpenDialog.Resize(fyne.NewSize(900, 500))
	fileOpenDialog.Show()
}

func (a *SpuWindow) readJson(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	mods := a.Modules
	if e := decoder.Decode(&mods); e != nil {
		return e
	}

	a.Modules = mods
	if a.Modules.MainModule == nil {
		a.Modules.MainModule = &Module{Name: a.langmap[a.Modules.CurrentLang][1]}
	}
	a.SelectModule(a.Modules.MainModule)
	return nil
}

func (a *SpuWindow) beginScenario() {
	if len(a.Modules.ChildModules) == 0 {
		return
	}
	if a.currentScenario != nil {
		dialog.ShowError(fmt.Errorf("%s", a.langmap[a.Modules.CurrentLang][16]), a.Window)
		return
	}
	a.ApplyModuleChanges()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

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

	scenario := NewScenario(a.Modules.MainModule.Content, t, a.Modules.ChildModules)
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

			if a.makePDF.do {
				fyne.Do(func() {

				})
			} else {
				dialog.ShowInformation(a.langmap[a.Modules.CurrentLang][17], a.langmap[a.Modules.CurrentLang][18], a.Window)
			}

		}
	}()

}

func (a *SpuWindow) interruptScenario() {
	if a.currentScenario == nil {
		dialog.ShowError(fmt.Errorf("%s", a.langmap[a.Modules.CurrentLang][19]), a.Window)
		return
	}
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.currentScenario = nil
	a.Elms.activity.Hide()
	dialog.ShowInformation(a.langmap[a.Modules.CurrentLang][20], a.langmap[a.Modules.CurrentLang][21], a.Window)
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
				a.Elms.title.Text = fmt.Sprintf("%s '%s'", a.langmap[a.Modules.CurrentLang][15], m.Name)
				a.Elms.title.Refresh()
				a.refreshModuleGui()
			} else {
				return
			}
		}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(500, 300))
	addmoduleDialog.Show()
}

func (a *SpuWindow) addModule() {
	a.ApplyModuleChanges()
	input := widget.NewMultiLineEntry()
	scroll := container.NewVScroll(input)
	scroll.ScrollToBottom()
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
			if b {
				if input.Text == "" {
					return
				}
				m := &Module{
					Name:    input.Text,
					Content: "",
					Output:  "",
				}
				a.Modules.ChildModules = append(a.Modules.ChildModules, m)
				a.Elms.modulesPanel.Add(a.createModuleButton(m))
				a.Elms.modulesPanel.Refresh()
				a.ApplyModuleChanges()
				a.SelectModule(m)
			} else {
				return
			}
		}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(500, 300))
	addmoduleDialog.Show()
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
	return widget.NewButton(m.Name,
		func() {
			a.ApplyModuleChanges()
			a.SelectModule(m)
		})
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
	if line == a.langmap[a.Modules.CurrentLang][27] && module == a.selectedModule {
		module.Output += line
	} else if line != a.langmap[a.Modules.CurrentLang][27] {
		module.Output += line
	}
	if module == a.selectedModule {
		a.Elms.ModuleOutputEntryMutex.Lock()
		defer a.Elms.ModuleOutputEntryMutex.Unlock()
		a.Elms.ModuleOutputEntry.Text += line
		a.Elms.ModuleOutputEntry.CursorRow = strings.LastIndexAny(module.Output, "\n")
		a.Elms.ModuleOutputEntry.Refresh()
	}
}

func (a *SpuWindow) PDFcreationWindow() {
	filesavedialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			a.makePDFFile(writer, err)
		}, a.Window)
	filesavedialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
	filesavedialog.Resize(fyne.NewSize(900, 500))
	filesavedialog.Show()
}

func (a *SpuWindow) makePDFFile(writer fyne.URIWriteCloser, err error) {
	if err != nil || writer == nil {
		return
	}

	path := writer.URI().Path()
	if filepath.Ext(path) != ".pdf" {
		defer os.Remove(path)
	}

	a.makePDF.pdfPath = path
	a.makePDF.pdfPath = strings.TrimSuffix(a.makePDF.pdfPath, filepath.Ext(a.makePDF.pdfPath))
	a.makePDF.pdfPath += ".pdf"

	if filepath.Base(a.makePDF.pdfPath) == ".pdf" {
		a.makePDF.pdfPath = strings.TrimSuffix(a.Profiles.Path, ".json") + time.Now().Format(" 02.01.2006 15-04-05") + a.makePDF.pdfPath
	}
	a.PDF()

}

//go:embed TimesNewRoman.ttf
var tnrFont []byte

//go:embed TimesNewRomanB.ttf
var tnrbFont []byte

func (a *SpuWindow) PDF() {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "", tnrFont)
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "B", tnrbFont)
	pdf.AddPage()
	pdf.SetFont("TimesNewRoman", "", 22)
	pdf.SetTextColor(0, 0, 0)
	for _, m := range a.Modules.ChildModules {
		pdf.SetFontSize(22)
		pdf.SetFontStyle("B")
		pdf.Cell(0, 10, m.Name)
		pdf.Ln(15)
		pdf.SetFontSize(14)
		pdf.SetFontStyle("")
		pdf.MultiCell(0, 10, m.Output, "0", "L", false)
	}
	e := pdf.OutputFileAndClose(a.makePDF.pdfPath)
	if e != nil {
		dialog.ShowError(fmt.Errorf("%s:\n%s", a.langmap[a.Modules.CurrentLang][28], e), a.Window)
	}
}

func (a *SpuWindow) changeLanguageWindow() {
	a.ApplyModuleChanges()
	options := []string{"English", "Русский"}
	dropoutMenu := widget.NewSelectEntry(options)
	langwindow := dialog.NewCustomConfirm(a.langmap[a.Modules.CurrentLang][29], a.langmap[a.Modules.CurrentLang][30], a.langmap[a.Modules.CurrentLang][24],
		container.NewBorder(
			container.NewVBox(canvas.NewText(a.langmap[a.Modules.CurrentLang][31], color.Black), dropoutMenu),
			nil, nil, nil,
		), func(b bool) {
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

func (a *SpuWindow) fullrefresh() {
	a.Modules.MainModule.Name = a.langmap[a.Modules.CurrentLang][0]

	a.Elms.topPanel.RemoveAll()
	a.Elms.topPanel.Add(
		utility.NewDropButton(theme.FolderOpenIcon(), a.Window.Canvas(), fyne.NewMenu("Profiles",
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][5], func() { a.loadProfile() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][6], func() { a.loadProfileInNewWindow() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][7], func() { a.saveProfile() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][8], func() { a.saveProfileAs() }),
		)))

	a.Elms.topPanel.Add(
		utility.NewDropButton(theme.MediaPlayIcon(), a.Window.Canvas(), fyne.NewMenu("Scenario",
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][9], func() { a.beginScenario() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][10], func() { a.interruptScenario() }),
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][11], func() {
				a.interruptScenario()
				a.PDFcreationWindow()
			}),
		)))

	a.Elms.topPanel.Add(
		utility.NewDropButton(theme.SettingsIcon(), a.Window.Canvas(), fyne.NewMenu("Change Language",
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][12], func() { a.changeLanguageWindow() }))))

	a.Elms.topPanel.Add(
		utility.NewDropButton(theme.CancelIcon(), a.Window.Canvas(), fyne.NewMenu("Quit",
			fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][13], func() { a.Window.Close() }),
		)))

	a.refreshModuleGui()

	a.Elms.title.Text = fmt.Sprintf("%s '%s'", a.langmap[a.Modules.CurrentLang][15], func(s string) string {
		if !strings.Contains(s, "\n") && len(s) < 30 {
			return s
		} else if len(s) > 30 {
			return s[:31] + "..."
		}
		s = strings.ReplaceAll(s, "\n", " ")
		return s[:31] + "..."
	}(a.selectedModule.Name))

	a.Elms.bottomPanelButtons.RemoveAll()
	a.Elms.bottomPanelButtons.Add(widget.NewButton(a.langmap[a.Modules.CurrentLang][3], func() { a.alter() }))
	a.Elms.bottomPanelButtons.Add(widget.NewButton(a.langmap[a.Modules.CurrentLang][4], func() { a.delete() }))

	a.Elms.threadEntry = widget.NewEntry()
	a.Elms.threadEntry.SetPlaceHolder(a.langmap[a.Modules.CurrentLang][1])
	a.Elms.threadEntry.Validator = utility.NumberValidator(1, 128)
	a.Elms.bottomPanelCheckboxes.RemoveAll()
	a.Elms.bottomPanelCheckboxes.Add(container.NewVBox(widget.NewCheck(a.langmap[a.Modules.CurrentLang][2], func(b bool) {
		a.makePDF.do = b
	})))
	a.Elms.mainButton.RemoveAll()
	a.Elms.bottomPanelCheckboxes.Add(a.Elms.threadEntry)
	a.Elms.mainButton.Add(widget.NewButton(a.langmap[a.Modules.CurrentLang][0], func() {
		a.ApplyModuleChanges()
		a.selectMainModule()
	}))
	a.Elms.addButton.RemoveAll()
	a.Elms.addButton.Add(widget.NewButton(a.langmap[a.Modules.CurrentLang][14], func() { a.addModule() }))
}
