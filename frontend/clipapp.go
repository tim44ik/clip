package frontend

import (
	"clip/config"
	"clip/engine/scenario"
	appErrors "clip/errors"
	"clip/locales"
	"clip/models/modules"
	"clip/processors/encrypter"
	"clip/processors/filemanager"
	"clip/processors/reporter"
	st "clip/processors/storage"
	"clip/utility"
	"context"
	_ "embed"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"gorm.io/gorm"
)

type ClipWindow struct {
	skip bool

	cancel context.CancelFunc

	modules *modules.ClipModules

	currentScenario *scenario.Scenario

	selectedModule *modules.Module

	encryptionType encrypter.Encrypter

	Window fyne.Window

	database *gorm.DB

	threads string

	langs    []string
	profiles struct {
		exists bool
		path   string
	}

	elms struct {
		moduleOutputEntryMutex sync.Mutex
		title                  *canvas.Text
		threadEntry            *widget.Entry
		moduleContentEntry     *widget.Entry
		moduleOutputEntry      *widget.Entry
		modulesPanel           *fyne.Container
		fullOutputContainer    *fyne.Container
		bottomPanelCheckboxes  *fyne.Container
		threadEntryBox         *fyne.Container
		editDeleteButtons      *fyne.Container
		topPanel               *fyne.Container
		activity               *widget.Activity
		mainButton             *fyne.Container
		bottomPanelButtons     *fyne.Container
	}
}

func CreateWindow() (a *ClipWindow) {
	db, err := config.Connect()
	if err != nil {
		log.Fatal("Failed to connect:", err)
		return
	}

	a = &ClipWindow{modules: &modules.ClipModules{}, database: db}

	err, a.langs = locales.Init()
	if err != nil {
		return
	}
	name := locales.T(a.modules.CurrentLang, "main")
	if name == "" {
		a.modules.CurrentLang = "en"
		name = locales.T(a.modules.CurrentLang, "main")
	}
	a.modules.MainModule = modules.CreateModule(name, "")

	a.profiles.exists = false

	a.buildWindow(fyne.CurrentApp())

	a.selectMainModule()

	a.fullRefresh()

	return
}

func (a *ClipWindow) buildWindow(app fyne.App) {
	a.Window = app.NewWindow("clip")
	a.elms.title = canvas.NewText("", color.Black)
	a.elms.title.TextSize = 16

	a.elms.moduleContentEntry = widget.NewMultiLineEntry()
	a.elms.moduleContentEntry.SetPlaceHolder("")

	a.elms.moduleOutputEntry = widget.NewMultiLineEntry()
	a.elms.moduleOutputEntry.Disable()

	a.elms.modulesPanel = container.NewVBox()

	a.elms.fullOutputContainer = container.NewVBox()

	a.elms.bottomPanelCheckboxes = container.NewVBox()
	a.elms.threadEntryBox = container.NewVBox()

	a.elms.editDeleteButtons = container.NewHBox()

	a.elms.activity = widget.NewActivity()

	a.elms.mainButton = container.NewVBox()

	a.elms.bottomPanelButtons = container.NewVBox()

	a.elms.topPanel = container.NewHBox()

	a.Window.SetContent(
		container.NewBorder(
			a.elms.topPanel,
			nil, nil, nil,
			container.NewBorder(
				nil,
				nil,
				container.NewBorder(
					nil,
					a.elms.bottomPanelButtons,
					nil,
					nil,

					container.NewVScroll(
						container.NewVBox(
							a.elms.mainButton,
							a.elms.modulesPanel,
						),
					),
				),
				nil,
				container.NewPadded(
					container.NewBorder(
						a.elms.title,
						container.NewBorder(nil, nil, container.NewBorder(nil, a.elms.activity, nil, nil),
							container.NewHBox(container.NewGridWithRows(1,
								container.NewGridWrap(
									fyne.NewSize(200, 15), a.elms.threadEntryBox,
								),
								a.elms.fullOutputContainer),
							),
						),
						nil,
						nil,
						container.NewGridWithRows(2, a.elms.moduleContentEntry, a.elms.moduleOutputEntry),
					),
				),
			),
		),
	)
	a.Window.Resize(fyne.NewSize(1280, 720))
	a.elms.activity.Hide()
}

func (a *ClipWindow) selectModule(m *modules.Module) {
	a.selectedModule = m
	a.elms.title.Text = fmt.Sprintf("%s '%s'", locales.T(a.modules.CurrentLang, "module"), a.formatTitle(m.Name))

	a.elms.title.Refresh()

	a.elms.moduleContentEntry.SetText(m.Content)

	a.elms.threadEntryBox.Hidden = m == a.modules.MainModule
	a.elms.fullOutputContainer.Hidden = m == a.modules.MainModule
	a.elms.moduleOutputEntry.Hidden = m == a.modules.MainModule
	a.elms.editDeleteButtons.Hidden = m == a.modules.MainModule

	a.cutOutput()
	a.elms.moduleOutputEntry.CursorRow = strings.LastIndexAny(a.elms.moduleOutputEntry.Text, "\n")
}

func (a *ClipWindow) applyModuleChanges() {
	if a.selectedModule == nil {
		return
	}
	a.selectedModule.Content = a.elms.moduleContentEntry.Text
	a.threads = a.elms.threadEntry.Text
}

func (a *ClipWindow) initCheck() bool {
	if len(a.modules.ChildModules) == 0 {
		return true
	}

	if a.currentScenario != nil {
		ShowError(a.modules.CurrentLang, appErrors.New(errScenarioInProcess), a.Window)
		return true
	}
	return false
}

func (a *ClipWindow) getThreads() (t int, err error) {
	if a.threads == "" {
		t = 1
	} else {
		t, err = strconv.Atoi(a.threads)
		if err != nil {
			return 0, appErrors.NewWithPlace(errDataFormat, appErrors.Place("threads_number"))
		}
	}
	return t, nil
}

func (a *ClipWindow) runner(scenario *scenario.Scenario, ctx context.Context) {
	a.elms.activity.Show()
	a.elms.activity.Start()

	errCh := make(chan error, len(a.modules.ChildModules)+1)
	defer close(errCh)
	go func() {
		for err := range errCh {
			if err != nil {
				ShowError(a.modules.CurrentLang, err, a.Window)
				a.interruptScenario()
			}
		}
	}()

	report := scenario.Execute(a.database, errCh, ctx,
		func(s any, m *modules.Module) {
			fyne.DoAndWait(func() { a.addModuleOutput(m, s) })
		})

	if a.currentScenario == scenario {
		fyne.DoAndWait(func() { a.elms.activity.Hide(); a.elms.activity.Stop() })
		a.currentScenario = nil
		if report == nil {
			dialog.ShowInformation(
				locales.T(a.modules.CurrentLang, "completed"),
				locales.T(a.modules.CurrentLang, "scenario_completed"),
				a.Window,
			)
		} else {
			go a.defineOutput(report, errCh, ctx)
		}
	}
}

func (a *ClipWindow) beginScenario() {
	if a.initCheck() {
		return
	}

	a.applyModuleChanges()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	t, err := a.getThreads()
	if err != nil {
		ShowError(a.modules.CurrentLang, err, a.Window)
		return
	}

	queue, err := utility.GetQueue(a.modules.ChildModules)
	if err != nil {
		ShowError(a.modules.CurrentLang, err, a.Window)
		return
	}

	scenario := scenario.NewScenario(a.modules.MainModule.Content, t, queue)
	a.currentScenario = scenario
	a.elms.moduleOutputEntry.Text = ""
	a.elms.moduleOutputEntry.Refresh()

	go a.runner(scenario, ctx)
}

func (a *ClipWindow) defineOutput(report *reporter.Report, errCh chan<- error, ctx context.Context) {
	f := filemanager.NewFileManager(
		a.modules.CurrentLang,
		a.profiles.path,
		a.Window,
		a.profiles.exists,
	)

	go f.ReportCreationWindow(report,
		func(path string) {

			if ctx.Err() != nil {
				return
			}

			progressChan := make(chan float64)
			defer close(progressChan)

			ext := report.Reporter.GetFileType()
			if filepath.Ext(path) != ext {
				defer os.Remove(path)
				path += ext
			}

			go report.Reporter.CreateReport(path, report.Content, errCh)
		},
	)
}

func (a *ClipWindow) interruptScenario() {
	if a.currentScenario == nil {
		ShowError(a.modules.CurrentLang, appErrors.New(errNotStarted), a.Window)
		return
	}
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.currentScenario = nil
	fyne.Do(func() { a.elms.activity.Hide() })

}

func (a *ClipWindow) selectMainModule() {
	a.selectModule(a.modules.MainModule)
}

func (a *ClipWindow) refreshModuleGui() {
	a.elms.modulesPanel.RemoveAll()
	for _, m := range a.modules.ChildModules {
		a.elms.modulesPanel.Add(a.createModuleButton(m))
	}

	a.elms.modulesPanel.Refresh()
	a.elms.bottomPanelButtons.Refresh()
	a.elms.mainButton.Refresh()
}

func (a *ClipWindow) addModuleOutput(module *modules.Module, line any) {
	switch line := line.(type) {
	case []interface{}:
		for i := range line {
			module.Output += fmt.Sprintf("\n%v", line[i])
		}
	default:
		module.Output += fmt.Sprintf("\n%v", line)
	}
	if module == a.selectedModule {
		a.elms.moduleOutputEntryMutex.Lock()
		defer a.elms.moduleOutputEntryMutex.Unlock()
		a.cutOutput()
		a.elms.moduleOutputEntry.CursorRow = strings.LastIndex(a.elms.moduleOutputEntry.Text, "\n")
		a.elms.moduleOutputEntry.Refresh()
	}
}

func (a *ClipWindow) cutOutput() {
	divided := strings.Split(a.selectedModule.Output, "\n")
	if len(divided) > 14 {
		a.elms.moduleOutputEntry.SetText(strings.Join(divided[len(divided)-15:], "\n"))
	} else {
		a.elms.moduleOutputEntry.SetText(a.selectedModule.Output)
	}
}

func (a *ClipWindow) formatTitle(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runeS := []rune(s)
	if len(runeS) < 30 {
		return s
	}
	return string(runeS[:31]) + "..."
}

func (a *ClipWindow) listenErrors(ctx context.Context, cancel context.CancelFunc, errChan chan error) {
	for {
		select {
		case <-ctx.Done():
			return

		case err := <-errChan:
			if err != nil {
				cancel()

				fyne.Do(func() {
					ShowError(a.modules.CurrentLang, err, a.Window)
				})
				close(errChan)
				return
			}
		}
	}
}

func (a *ClipWindow) fullRefresh() {
	a.elms.topPanel.RemoveAll()
	a.elms.topPanel.Add(
		utility.NewDropButton(theme.FolderOpenIcon(),
			a.Window.Canvas(), fyne.NewMenu("Profiles",
				fyne.NewMenuItem(locales.T(a.modules.CurrentLang, "load_script"),
					func() {
						a.applyModuleChanges()
						errChan := make(chan error)
						ctx, cancel := context.WithCancel(context.Background())

						go a.listenErrors(ctx, cancel, errChan)

						f := filemanager.NewFileManager(
							a.modules.CurrentLang,
							a.profiles.path,
							a.Window,
							a.profiles.exists,
						)

						f.LoadScripts(errChan, func(scripts []*filemanager.Script) {
							if ctx.Err() != nil {
								return
							}
							q, err := utility.GetQueue(a.modules.ChildModules)

							if err != nil {
								ShowError(a.modules.CurrentLang, err, a.Window)
							}
							f.OpenScriptPicker(scripts, func(name, content string) {
								if ctx.Err() != nil {
									return
								}

								a.add(name, strconv.Itoa(len(q))+content)
							})
						})
					},
				),
				fyne.NewMenuItem(locales.T(a.modules.CurrentLang, "load"),
					func() {
						a.applyModuleChanges()
						errChan := make(chan error)
						ctx, cancel := context.WithCancel(context.Background())

						go a.listenErrors(ctx, cancel, errChan)

						f := filemanager.NewFileManager(
							a.modules.CurrentLang,
							a.profiles.path,
							a.Window,
							a.profiles.exists,
						)

						f.LoadProfile(
							errChan,
							func(mods modules.ClipModules, path string, enc encrypter.Encrypter) {

								if ctx.Err() != nil {
									return
								}

								fyne.Do(func() {
									a.modules = &mods
									a.profiles.path = path
									a.encryptionType = enc
									a.profiles.exists = true

									a.refreshModuleGui()
									a.fullRefresh()
									a.selectModule(a.modules.MainModule)

									a.skip = true
								})
							},
						)
					}),
				fyne.NewMenuItem(locales.T(a.modules.CurrentLang, "save"),
					func() {
						a.applyModuleChanges()

						errChan := make(chan error)

						ctx, cancel := context.WithCancel(context.Background())

						go a.listenErrors(ctx, cancel, errChan)

						f := filemanager.NewFileManager(
							a.modules.CurrentLang,
							a.profiles.path,
							a.Window,
							a.profiles.exists,
						)

						f.GetProfileType(errChan, a.skip, func(e st.Encoder) {
							if ctx.Err() != nil {
								return
							}

							f.SaveProfile(errChan,
								a.encryptionType,
								e,
								a.modules,
								func(enc encrypter.Encrypter, exists bool, path string) {
									if ctx.Err() != nil {
										return
									}

									a.encryptionType = enc
									a.profiles.exists = exists
									a.profiles.path = path
									a.skip = true
								},
							)
						})
					},
				),
				fyne.NewMenuItem(locales.T(a.modules.CurrentLang, "save_as"),
					func() {
						a.applyModuleChanges()

						errChan := make(chan error)

						ctx, cancel := context.WithCancel(context.Background())

						go a.listenErrors(ctx, cancel, errChan)

						f := filemanager.NewFileManager(
							a.modules.CurrentLang,
							a.profiles.path,
							a.Window,
							a.profiles.exists,
						)

						f.GetProfileType(errChan, false, func(e st.Encoder) {

							if ctx.Err() != nil {
								return
							}

							f.GetEncryptionType(func(enc encrypter.Encrypter) {

								if ctx.Err() != nil {
									return
								}

								if enc == nil {
									f.SaveProfileAs(errChan,
										"",
										enc,
										e,
										a.modules,
										func(enc encrypter.Encrypter, exists bool, path string) {

											if ctx.Err() != nil {
												return
											}

											a.encryptionType = enc
											a.profiles.exists = exists
											a.profiles.path = path
											a.skip = true
										},
									)
									return
								}

								f.GetPassword(errChan, func(p string) {

									if ctx.Err() != nil {
										return
									}

									f.SaveProfileAs(
										errChan,
										p,
										enc,
										e,
										a.modules,
										func(enc encrypter.Encrypter, exists bool, path string) {

											if ctx.Err() != nil {
												return
											}

											a.encryptionType = enc
											a.profiles.exists = exists
											a.profiles.path = path
											a.skip = true
										},
									)
								})
							})
						})
					},
				),
			)))

	a.elms.topPanel.Add(
		utility.NewDropButton(theme.MediaPlayIcon(),
			a.Window.Canvas(), fyne.NewMenu("Scenario",
				fyne.NewMenuItem(locales.T(a.modules.CurrentLang, "begin_scenario"),
					func() { a.beginScenario() }),
				fyne.NewMenuItem(locales.T(a.modules.CurrentLang, "break_scenario"),
					func() { a.interruptScenario() }),
			)))

	a.elms.topPanel.Add(
		utility.NewDropButton(theme.SettingsIcon(),
			a.Window.Canvas(), fyne.NewMenu("Change Language",
				fyne.NewMenuItem(locales.T(a.modules.CurrentLang, "change_language"),
					func() { a.changeLanguageWindow() }))))

	a.modules.MainModule.Name = locales.T(a.modules.CurrentLang, "main")
	a.elms.mainButton.RemoveAll()
	a.elms.mainButton.Add(widget.NewButton(
		a.modules.MainModule.Name, func() {
			a.applyModuleChanges()
			a.selectMainModule()
		}))

	a.elms.title.Text = fmt.Sprintf("%s '%s'",
		locales.T(a.modules.CurrentLang, "module"), a.formatTitle(a.selectedModule.Name))

	a.elms.fullOutputContainer.RemoveAll()
	a.elms.fullOutputContainer.Add(container.NewVBox(
		widget.NewButton(locales.T(a.modules.CurrentLang, "view_full_output"),
			func() { a.fullOutput() })))

	a.elms.editDeleteButtons.RemoveAll()
	a.elms.editDeleteButtons.Add(widget.NewButton(
		locales.T(a.modules.CurrentLang, "edit"),
		func() { a.editModuleName() }))
	a.elms.editDeleteButtons.Add(widget.NewButton(
		locales.T(a.modules.CurrentLang, "delete"),
		func() { a.deleteModule() }))

	a.elms.bottomPanelButtons.RemoveAll()
	a.elms.bottomPanelButtons.Add(widget.NewButton(
		locales.T(a.modules.CurrentLang, "add_module"),
		func() { a.addDialog() }))
	a.elms.bottomPanelButtons.Add(a.elms.editDeleteButtons)

	a.elms.threadEntry = widget.NewEntry()
	a.elms.threadEntry.SetPlaceHolder(
		locales.T(a.modules.CurrentLang, "threads_number"),
	)
	a.elms.threadEntry.Text = a.threads

	a.elms.threadEntryBox.RemoveAll()
	a.elms.threadEntryBox.Add(a.elms.threadEntry)
}
