package core

import (
	"clip/errors"
	"clip/modules"

	"encoding/json"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

func LoadProfileInNewWindow(a *ClipWindow) {
	fileOpenDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {
			loadInNewWindowDialogTrue(a, reader, errors.UniversalError{ErrorText: a.langmap[a.Modules.CurrentLang][40]})
		},
		a.Window,
	)

	fileOpenDialog.SetFilter(
		storage.NewExtensionFileFilter([]string{".json"}),
	)
	fileOpenDialog.Resize(fyne.NewSize(900, 500))
	fileOpenDialog.Show()
}

func loadInNewWindowDialogTrue(a *ClipWindow, reader fyne.URIReadCloser, err error) {
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

	err = readJson(newWindow, filename)
	if err != nil {
		dialog.ShowError(err, a.Window)
		return
	}

	newWindow.profiles.exists = true
	newWindow.profiles.path = filename
	newWindow.Window.Show()

	newWindow.refreshModuleGui()
	newWindow.fullrefresh()
}

func LoadProfile(a *ClipWindow) {
	fileOpenDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {
			loadDialogTrue(a, reader, err)
		},
		a.Window,
	)

	fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	fileOpenDialog.Resize(fyne.NewSize(900, 500))
	fileOpenDialog.Show()
}

func loadDialogTrue(a *ClipWindow, reader fyne.URIReadCloser, err error) {
	if err != nil {
		dialog.ShowError(err, a.Window)
		return
	}
	if reader == nil {
		return
	}

	filename := reader.URI().Path()
	reader.Close()

	err = readJson(a, filename)
	if err != nil {
		dialog.ShowError(err, a.Window)
		return
	}

	a.profiles.exists = true
	a.profiles.path = filename

	a.refreshModuleGui()
	a.fullrefresh()
}

func readJson(a *ClipWindow, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.UniversalError{ErrorText: a.langmap[a.Modules.CurrentLang][40]}
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	mods := a.Modules
	if e := decoder.Decode(&mods); e != nil {
		return errors.UniversalError{ErrorText: a.langmap[a.Modules.CurrentLang][40]}
	}

	a.Modules = mods
	if a.Modules.MainModule == nil {
		a.Modules.MainModule = &modules.Module{
			Name: a.langmap[a.Modules.CurrentLang][1],
		}
	}
	a.selectModule(a.Modules.MainModule)
	return nil
}
