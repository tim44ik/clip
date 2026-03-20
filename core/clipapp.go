package core

import (
	"clip/encrypter"
	"clip/errors"
	"clip/filemanager"
	"clip/modules"
	outputprocessor "clip/outputProcessor"
	"clip/reporter"
	"clip/scenario"
	st "clip/storage"
	"clip/utility"
	"context"
	_ "embed"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ClipWindow struct {
	skip bool

	cancel context.CancelFunc

	modules *modules.ClipModules

	currentScenario *scenario.Scenario

	selectedModule *modules.Module

	langmap map[string][]string

	encryptionType encrypter.Encrypter

	Window fyne.Window

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
		createReportCheck      *widget.Check
		processOutputCheck     *widget.Check
		modulesPanel           *fyne.Container
		fullOutputContainer    *fyne.Container
		threadEntryBox         *fyne.Container
		bottomPanelCheckboxes  *fyne.Container
		editDeleteButtons      *fyne.Container
		topPanel               *fyne.Container
		activity               *widget.Activity
		mainButton             *fyne.Container
		bottomPanelButtons     *fyne.Container
	}
}

func CreateWindow() (a *ClipWindow) {
	a = &ClipWindow{modules: &modules.ClipModules{}}
	a.langmapInit()
	_, ok := a.langmap[a.modules.CurrentLang]
	if !ok {
		a.modules.CurrentLang = "en"
	}

	a.modules.MainModule = &modules.Module{Name: a.langmap[a.modules.CurrentLang][0]}

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
							container.NewHBox(
								container.NewCenter(
									container.NewGridWrap(
										fyne.NewSize(200, 15), a.elms.threadEntryBox,
									),
								), a.elms.bottomPanelCheckboxes, a.elms.fullOutputContainer,
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
	a.Window.Resize(fyne.NewSize(900, 600))
	a.Window.SetOnClosed(func() {
		a.interruptScenario()
		a.Window.Close()
	})
	a.elms.activity.Hide()
}

func (a *ClipWindow) selectModule(m *modules.Module) {
	a.selectedModule = m
	a.elms.title.Text = fmt.Sprintf("%s '%s'", a.langmap[a.modules.CurrentLang][15], func(s string) string {
		if !strings.Contains(s, "\n") && len(s) < 30 {
			return s
		} else if strings.Contains(s, "\n") && len(s) < 30 {
			return strings.ReplaceAll(s, "\n", " ")
		}
		s = strings.ReplaceAll(s, "\n", " ")
		return s[:31] + " ..."
	}(m.Name))

	a.elms.title.Refresh()

	a.elms.moduleContentEntry.SetText(m.Content)

	if a.elms.createReportCheck != nil && a.elms.processOutputCheck != nil {
		if a.selectedModule == a.modules.MainModule {
			for _, m := range a.modules.ChildModules {
				if m.MakeReport.Do {
					a.elms.createReportCheck.Checked = true
					break
				}
				a.elms.createReportCheck.Checked = false
			}
			if a.elms.createReportCheck.Checked {
				for _, m := range a.modules.ChildModules {
					if m.MakeReport.Process {
						a.elms.processOutputCheck.Checked = true
						break
					}
					a.elms.processOutputCheck.Checked = false
				}
			}
		} else {
			if a.selectedModule.MakeReport.Do && a.selectedModule.MakeReport.Process {
				a.elms.createReportCheck.Checked = true
				a.elms.processOutputCheck.Checked = true
				a.elms.processOutputCheck.Enable()
			} else if a.selectedModule.MakeReport.Do && !a.selectedModule.MakeReport.Process {
				a.elms.createReportCheck.Checked = true
				a.elms.processOutputCheck.Checked = false
				a.elms.processOutputCheck.Enable()
			} else {
				a.elms.createReportCheck.Checked = false
				a.elms.processOutputCheck.Checked = false
				a.selectedModule.MakeReport.Process = false
			}
		}
	}

	a.elms.fullOutputContainer.Hidden = m == a.modules.MainModule
	a.elms.moduleOutputEntry.Hidden = m == a.modules.MainModule
	a.elms.editDeleteButtons.Hidden = m == a.modules.MainModule

	a.cutOutput()
	a.elms.moduleOutputEntry.CursorRow = strings.LastIndexAny(a.elms.moduleOutputEntry.Text, "\n")
	a.elms.bottomPanelCheckboxes.Refresh()
}

func (a *ClipWindow) applyModuleChanges() {
	if a.selectedModule == nil {
		return
	}
	a.selectedModule.Content = a.elms.moduleContentEntry.Text
}

func (a *ClipWindow) initCheck() bool {
	if len(a.modules.ChildModules) == 0 {
		return true
	}

	if a.currentScenario != nil {
		dialog.ShowError(fmt.Errorf("%s", a.langmap[a.modules.CurrentLang][16]), a.Window)
		return true
	}
	return false
}

func (a *ClipWindow) getThreads() (t int, err error) {
	if a.elms.threadEntry.Text == "" {
		t = 1
	} else {
		t, err = strconv.Atoi(a.elms.threadEntry.Text)
		if err != nil {
			return 0, errors.UniversalError{ErrorText: a.langmap[a.modules.CurrentLang][39], Module: a.langmap[a.modules.CurrentLang][2]}
		}
	}
	return t, nil
}

func (a *ClipWindow) runner(scenario *scenario.Scenario, ctx context.Context) {
	a.elms.activity.Show()
	a.elms.activity.Start()

	scenario.BeginScenario(ctx,
		func(s string, m *modules.Module) {
			fyne.DoAndWait(func() { a.addModuleOutput(m, s) })
		})

	if a.currentScenario == scenario {
		fyne.DoAndWait(func() { a.elms.activity.Hide(); a.elms.activity.Stop() })
		a.currentScenario = nil

		if !a.defineOutput(ctx) {
			dialog.ShowInformation(
				a.langmap[a.modules.CurrentLang][17],
				a.langmap[a.modules.CurrentLang][18],
				a.Window,
			)
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
		dialog.ShowError(err, a.Window)
		return
	}

	queue, err := utility.GetQueue(a.langmap[a.modules.CurrentLang], a.modules.ChildModules)
	if err != nil {
		dialog.ShowError(err, a.Window)
		return
	}

	scenario := scenario.NewScenario(a.modules.MainModule.Content, t, queue)
	a.currentScenario = scenario
	a.elms.moduleOutputEntry.Text = ""
	a.elms.moduleOutputEntry.Refresh()

	go a.runner(scenario, ctx)
}

func (a *ClipWindow) defineOutput(ctx context.Context) bool {
	makeReportFor := []*modules.Module{}
	for _, m := range a.modules.ChildModules {
		if m.MakeReport.Do {
			makeReportFor = append(makeReportFor, m)
		}
	}

	if len(makeReportFor) > 0 {
		f := filemanager.NewFileManager(a.Window, a.langmap[a.modules.CurrentLang], a.profiles.path, a.profiles.exists)
		f.GetReportType(func(r reporter.Reporter) {
			f.GetDBType(ctx, func(db outputprocessor.DB) {
				go f.ReportСreationWindow(db, r, makeReportFor)
			})

		})
		return true
	}
	return false
}

func (a *ClipWindow) interruptScenario() {
	if a.currentScenario == nil {
		dialog.ShowError(errors.UniversalError{ErrorText: a.langmap[a.modules.CurrentLang][19], Module: ""}, a.Window)
		return
	}
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.currentScenario = nil
	a.elms.activity.Hide()
	dialog.ShowInformation(
		a.langmap[a.modules.CurrentLang][20],
		a.langmap[a.modules.CurrentLang][21],
		a.Window,
	)
}

func (a *ClipWindow) selectMainModule() {
	a.selectModule(a.modules.MainModule)
}

func (a *ClipWindow) refreshModuleGui() {
	a.elms.modulesPanel.RemoveAll()
	for _, m := range a.modules.ChildModules {
		a.elms.modulesPanel.Add(CreateModuleButton(a, m))
	}

	a.elms.modulesPanel.Refresh()
	a.elms.bottomPanelButtons.Refresh()
	a.elms.mainButton.Refresh()
}

func (a *ClipWindow) addModuleOutput(module *modules.Module, line string) {
	module.Output += line
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

func (a *ClipWindow) fullRefresh() {
	a.elms.topPanel.RemoveAll()
	a.elms.topPanel.Add(
		utility.NewDropButton(theme.FolderOpenIcon(),
			a.Window.Canvas(), fyne.NewMenu("Profiles",
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][5],
					func() {
						a.applyModuleChanges()
						f := filemanager.NewFileManager(a.Window,
							a.langmap[a.modules.CurrentLang],
							a.profiles.path,
							a.profiles.exists)
						f.LoadProfile(
							func(mods modules.ClipModules, path string, enc encrypter.Encrypter) {
								a.modules = &mods
								a.profiles.path = path
								a.encryptionType = enc
								a.profiles.exists = true
								a.refreshModuleGui()
								a.fullRefresh()
								a.selectModule(a.modules.MainModule)
								a.skip = true
							},
						)
					}),
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][6],
					func() {
						a.applyModuleChanges()
						f := filemanager.NewFileManager(a.Window,
							a.langmap[a.modules.CurrentLang],
							a.profiles.path,
							a.profiles.exists)
						f.LoadProfile(func(mods modules.ClipModules, path string, enc encrypter.Encrypter) {
							newWindow := CreateWindow()
							newWindow.modules = &mods
							newWindow.profiles.path = path
							newWindow.encryptionType = enc
							newWindow.profiles.exists = true
							newWindow.skip = true
							newWindow.refreshModuleGui()
							newWindow.fullRefresh()
							newWindow.selectModule(newWindow.modules.MainModule)
							newWindow.Window.Show()
						})
					}),
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][7],
					func() {
						a.applyModuleChanges()
						f := filemanager.NewFileManager(a.Window,
							a.langmap[a.modules.CurrentLang],
							a.profiles.path,
							a.profiles.exists)
						f.GetProfileType(a.skip, func(e st.Encoder) {
							f.SaveProfile(a.encryptionType, e, a.modules,
								func(e encrypter.Encrypter, exists bool, path string) {
									a.encryptionType = e
									a.profiles.exists = exists
									a.profiles.path = path
									a.skip = true
								})
						})

					}),
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][8],
					func() {
						a.applyModuleChanges()
						f := filemanager.NewFileManager(a.Window,
							a.langmap[a.modules.CurrentLang],
							a.profiles.path,
							a.profiles.exists)
						f.GetProfileType(false, func(e st.Encoder) {
							f.GetEncryptionType(func(enc encrypter.Encrypter) {
								if enc == nil {
									f.SaveProfileAs("", enc, e, a.modules,
										func(enc encrypter.Encrypter, exists bool, path string) {
											a.encryptionType = enc
											a.profiles.exists = exists
											a.profiles.path = path
											a.skip = true

										},
									)
									return
								}
								f.GetPassword(func(p string) {
									f.SaveProfileAs(p, enc, e, a.modules,
										func(enc encrypter.Encrypter, exists bool, path string) {
											a.encryptionType = enc
											a.profiles.exists = exists
											a.profiles.path = path
											a.skip = true
										},
									)
								})
							})
						})

					}),
			)))

	a.elms.topPanel.Add(
		utility.NewDropButton(theme.MediaPlayIcon(),
			a.Window.Canvas(), fyne.NewMenu("Scenario",
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][9],
					func() { a.beginScenario() }),
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][10],
					func() { a.interruptScenario() }),
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][11],
					func() {
						a.interruptScenario()
						a.defineOutput(context.Background())
					},
				),
			)))

	a.elms.topPanel.Add(
		utility.NewDropButton(theme.SettingsIcon(),
			a.Window.Canvas(), fyne.NewMenu("Change Language",
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][12],
					func() { a.changeLanguageWindow() }))))

	a.elms.topPanel.Add(
		utility.NewDropButton(theme.CancelIcon(),
			a.Window.Canvas(), fyne.NewMenu("Quit",
				fyne.NewMenuItem(a.langmap[a.modules.CurrentLang][13],
					func() { a.Window.Close() }),
			)))

	a.modules.MainModule.Name = a.langmap[a.modules.CurrentLang][0]
	a.elms.mainButton.RemoveAll()
	a.elms.mainButton.Add(widget.NewButton(
		a.langmap[a.modules.CurrentLang][0], func() {
			a.applyModuleChanges()
			a.selectMainModule()
		}))

	a.elms.title.Text = fmt.Sprintf("%s '%s'",
		a.langmap[a.modules.CurrentLang][15],
		func(s string) string {
			if !strings.Contains(s, "\n") && len(s) < 30 {
				return s
			} else if strings.Contains(s, "\n") && len(s) < 30 {
				return strings.ReplaceAll(s, "\n", " ")
			}
			s = strings.ReplaceAll(s, "\n", " ")
			return s[:31] + " ..."
		}(a.selectedModule.Name))

	a.elms.fullOutputContainer.RemoveAll()
	a.elms.fullOutputContainer.Add(container.NewVBox(
		widget.NewButton(a.langmap[a.modules.CurrentLang][32],
			func() { a.fullOutput() })))

	a.elms.editDeleteButtons.RemoveAll()
	a.elms.editDeleteButtons.Add(widget.NewButton(
		a.langmap[a.modules.CurrentLang][3],
		func() { a.editModuleName() }))
	a.elms.editDeleteButtons.Add(widget.NewButton(
		a.langmap[a.modules.CurrentLang][4],
		func() { a.deleteModule() }))

	a.elms.bottomPanelButtons.RemoveAll()
	a.elms.bottomPanelButtons.Add(widget.NewButton(
		a.langmap[a.modules.CurrentLang][14],
		func() { a.addModule() }))
	a.elms.bottomPanelButtons.Add(a.elms.editDeleteButtons)

	a.elms.threadEntry = widget.NewEntry()
	a.elms.threadEntry.SetPlaceHolder(
		a.langmap[a.modules.CurrentLang][1],
	)

	a.elms.processOutputCheck = widget.NewCheck(
		a.langmap[a.modules.CurrentLang][34],
		func(b bool) {
			a.selectedModule.MakeReport.Process = b
			if a.selectedModule == a.modules.MainModule && a.elms.createReportCheck.Checked {
				for _, m := range a.modules.ChildModules {
					m.MakeReport.Process = b
				}
			}
		})

	a.elms.createReportCheck = widget.NewCheck(
		a.langmap[a.modules.CurrentLang][2],
		func(b bool) {
			a.selectedModule.MakeReport.Do = b
			if !a.selectedModule.MakeReport.Do {
				a.elms.processOutputCheck.SetChecked(false)
				a.elms.processOutputCheck.Disable()
			} else {
				a.elms.processOutputCheck.Enable()
			}
			if a.selectedModule == a.modules.MainModule {
				for _, m := range a.modules.ChildModules {
					m.MakeReport.Do = b
				}
			}
		})

	a.elms.createReportCheck.Checked = a.selectedModule.MakeReport.Do
	a.elms.processOutputCheck.Checked = a.selectedModule.MakeReport.Process

	a.elms.bottomPanelCheckboxes.RemoveAll()
	a.elms.bottomPanelCheckboxes.Add(a.elms.createReportCheck)
	a.elms.bottomPanelCheckboxes.Add(a.elms.processOutputCheck)

	a.elms.threadEntryBox.RemoveAll()
	a.elms.threadEntryBox.Add(a.elms.threadEntry)
}
