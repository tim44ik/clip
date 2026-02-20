package core

import (
	"clip/errors"
	"clip/modules"
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
	Window fyne.Window

	cancel context.CancelFunc

	elms struct {
		moduleOutputEntryMutex sync.Mutex
		title                  *canvas.Text
		threadEntry            *widget.Entry
		moduleContentEntry     *widget.Entry
		moduleOutputEntry      *widget.Entry
		createPDFCheck         *widget.Check
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

	currentScenario *Scenario

	selectedModule *modules.Module

	langmap map[string][]string

	profiles struct {
		exists bool
		path   string
	}

	Modules struct {
		MainModule   *modules.Module   `json:"mainModule"`
		ChildModules []*modules.Module `json:"childModules"`
		CurrentLang  string            `json:"currentLang"`
	}
}

func CreateWindow() (a *ClipWindow) {
	a = &ClipWindow{}
	LangmapInit(a)
	_, ok := a.langmap[a.Modules.CurrentLang]
	if !ok {
		a.Modules.CurrentLang = "en"
	}

	a.Modules.MainModule = &modules.Module{Name: a.langmap[a.Modules.CurrentLang][0]}

	a.profiles.exists = false

	a.buildWindow(fyne.CurrentApp())

	a.selectMainModule()

	a.fullrefresh()

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
	a.elms.title.Text = fmt.Sprintf("%s '%s'", a.langmap[a.Modules.CurrentLang][15], func(s string) string {
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

	if a.elms.createPDFCheck != nil && a.elms.processOutputCheck != nil {
		if a.selectedModule == a.Modules.MainModule {
			for _, m := range a.Modules.ChildModules {
				if m.MakePDF.Do {
					a.elms.createPDFCheck.Checked = true
					break
				}
				a.elms.createPDFCheck.Checked = false
			}
			if a.elms.createPDFCheck.Checked {
				for _, m := range a.Modules.ChildModules {
					if m.MakePDF.Process {
						a.elms.processOutputCheck.Checked = true
						break
					}
					a.elms.processOutputCheck.Checked = false
				}
			}
		} else {
			if a.selectedModule.MakePDF.Do && a.selectedModule.MakePDF.Process {
				a.elms.createPDFCheck.Checked = true
				a.elms.processOutputCheck.Checked = true
				a.elms.processOutputCheck.Enable()
			} else if a.selectedModule.MakePDF.Do && !a.selectedModule.MakePDF.Process {
				a.elms.createPDFCheck.Checked = true
				a.elms.processOutputCheck.Checked = false
				a.elms.processOutputCheck.Enable()
			} else {
				a.elms.createPDFCheck.Checked = false
				a.elms.processOutputCheck.Checked = false
				a.selectedModule.MakePDF.Process = false
			}
		}
	}

	a.elms.fullOutputContainer.Hidden = m == a.Modules.MainModule
	a.elms.moduleOutputEntry.Hidden = m == a.Modules.MainModule
	a.elms.editDeleteButtons.Hidden = m == a.Modules.MainModule

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
	if len(a.Modules.ChildModules) == 0 {
		return true
	}

	if a.currentScenario != nil {
		dialog.ShowError(fmt.Errorf("%s", a.langmap[a.Modules.CurrentLang][16]), a.Window)
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
			return 0, errors.UniversalError{ErrorText: a.langmap[a.Modules.CurrentLang][39], Module: a.langmap[a.Modules.CurrentLang][2]}
		}
	}
	return t, nil
}

func (a *ClipWindow) runner(scenario *Scenario, ctx context.Context) {
	a.elms.activity.Show()
	a.elms.activity.Start()

	scenario.BeginScenario(ctx,
		func(s string, m *modules.Module) {
			fyne.DoAndWait(func() { a.addModuleOutput(m, s) })
		})

	if a.currentScenario == scenario {
		fyne.DoAndWait(func() { a.elms.activity.Hide(); a.elms.activity.Stop() })
		a.currentScenario = nil

		if !a.filterPDF(ctx) {
			dialog.ShowInformation(
				a.langmap[a.Modules.CurrentLang][17],
				a.langmap[a.Modules.CurrentLang][18],
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

	queue, err := utility.GetQueue(a.langmap[a.Modules.CurrentLang], a.Modules.ChildModules)
	if err != nil {
		dialog.ShowError(err, a.Window)
		return
	}

	scenario := NewScenario(a.Modules.MainModule.Content, t, queue)
	a.currentScenario = scenario
	a.elms.moduleOutputEntry.Text = ""
	a.elms.moduleOutputEntry.Refresh()

	go a.runner(scenario, ctx)
}

func (a *ClipWindow) filterPDF(ctx context.Context) bool {
	makePDFFor := []*modules.Module{}
	for _, m := range a.Modules.ChildModules {
		if m.MakePDF.Do {
			makePDFFor = append(makePDFFor, m)
		}
	}
	if len(makePDFFor) > 0 {
		go PDFcreationWindow(a, makePDFFor, ctx)
		return true
	}
	return false
}

func (a *ClipWindow) interruptScenario() {
	if a.currentScenario == nil {
		dialog.ShowError(errors.UniversalError{ErrorText: a.langmap[a.Modules.CurrentLang][19], Module: ""}, a.Window)
		return
	}
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.currentScenario = nil
	a.elms.activity.Hide()
	dialog.ShowInformation(
		a.langmap[a.Modules.CurrentLang][20],
		a.langmap[a.Modules.CurrentLang][21],
		a.Window,
	)
}

func (a *ClipWindow) selectMainModule() {
	a.selectModule(a.Modules.MainModule)
}

func (a *ClipWindow) refreshModuleGui() {
	a.elms.modulesPanel.RemoveAll()
	for _, m := range a.Modules.ChildModules {
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

func (a *ClipWindow) fullrefresh() {
	a.elms.topPanel.RemoveAll()
	a.elms.topPanel.Add(
		utility.NewDropButton(theme.FolderOpenIcon(),
			a.Window.Canvas(), fyne.NewMenu("Profiles",
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][5],
					func() {
						LoadProfile(a)
					}),
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][6],
					func() {
						LoadProfileInNewWindow(a)
					}),
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][7],
					func() { SaveProfile(a) }),
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][8],
					func() { SaveProfileAs(a) }),
			)))

	a.elms.topPanel.Add(
		utility.NewDropButton(theme.MediaPlayIcon(),
			a.Window.Canvas(), fyne.NewMenu("Scenario",
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][9],
					func() { a.beginScenario() }),
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][10],
					func() { a.interruptScenario() }),
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][11],
					func() {
						a.interruptScenario()
						a.filterPDF(context.Background())
					},
				),
			)))

	a.elms.topPanel.Add(
		utility.NewDropButton(theme.SettingsIcon(),
			a.Window.Canvas(), fyne.NewMenu("Change Language",
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][12],
					func() { ChangeLanguageWindow(a) }))))

	a.elms.topPanel.Add(
		utility.NewDropButton(theme.CancelIcon(),
			a.Window.Canvas(), fyne.NewMenu("Quit",
				fyne.NewMenuItem(a.langmap[a.Modules.CurrentLang][13],
					func() { a.Window.Close() }),
			)))

	a.Modules.MainModule.Name = a.langmap[a.Modules.CurrentLang][0]
	a.elms.mainButton.RemoveAll()
	a.elms.mainButton.Add(widget.NewButton(
		a.langmap[a.Modules.CurrentLang][0], func() {
			a.applyModuleChanges()
			a.selectMainModule()
		}))

	a.elms.title.Text = fmt.Sprintf("%s '%s'",
		a.langmap[a.Modules.CurrentLang][15],
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
		widget.NewButton(a.langmap[a.Modules.CurrentLang][32],
			func() { FullOutput(a) })))

	a.elms.editDeleteButtons.RemoveAll()
	a.elms.editDeleteButtons.Add(widget.NewButton(
		a.langmap[a.Modules.CurrentLang][3],
		func() { EditModuleName(a) }))
	a.elms.editDeleteButtons.Add(widget.NewButton(
		a.langmap[a.Modules.CurrentLang][4],
		func() { Delete(a) }))

	a.elms.bottomPanelButtons.RemoveAll()
	a.elms.bottomPanelButtons.Add(widget.NewButton(
		a.langmap[a.Modules.CurrentLang][14],
		func() { AddModule(a) }))
	a.elms.bottomPanelButtons.Add(a.elms.editDeleteButtons)

	a.elms.threadEntry = widget.NewEntry()
	a.elms.threadEntry.SetPlaceHolder(
		a.langmap[a.Modules.CurrentLang][1],
	)

	a.elms.processOutputCheck = widget.NewCheck(
		a.langmap[a.Modules.CurrentLang][34],
		func(b bool) {
			a.selectedModule.MakePDF.Process = b
			if a.selectedModule == a.Modules.MainModule && a.elms.createPDFCheck.Checked {
				for _, m := range a.Modules.ChildModules {
					m.MakePDF.Process = b
				}
			}
		})

	a.elms.createPDFCheck = widget.NewCheck(
		a.langmap[a.Modules.CurrentLang][2],
		func(b bool) {
			a.selectedModule.MakePDF.Do = b
			if !a.selectedModule.MakePDF.Do {
				a.elms.processOutputCheck.SetChecked(false)
				a.elms.processOutputCheck.Disable()
			} else {
				a.elms.processOutputCheck.Enable()
			}
			if a.selectedModule == a.Modules.MainModule {
				for _, m := range a.Modules.ChildModules {
					m.MakePDF.Do = b
				}
			}
		})

	a.elms.createPDFCheck.Checked = a.selectedModule.MakePDF.Do
	a.elms.processOutputCheck.Checked = a.selectedModule.MakePDF.Process

	a.elms.bottomPanelCheckboxes.RemoveAll()
	a.elms.bottomPanelCheckboxes.Add(a.elms.createPDFCheck)
	a.elms.bottomPanelCheckboxes.Add(a.elms.processOutputCheck)

	a.elms.threadEntryBox.RemoveAll()
	a.elms.threadEntryBox.Add(a.elms.threadEntry)

}
