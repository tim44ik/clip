package core

import (
	"encoding/json"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

func LoadProfileInNewWindow(a *SpuWindow) {
	fileOpenDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {
			loadInNewWindowDialogTrue(a, reader, err)
		},
		a.Window,
	)

	fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	fileOpenDialog.Resize(fyne.NewSize(900, 500))
	fileOpenDialog.Show()
}
func loadInNewWindowDialogTrue(a *SpuWindow, reader fyne.URIReadCloser, err error) {
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

	newWindow.Profiles.Exists = true
	newWindow.Profiles.Path = filename
	newWindow.Window.Show()
	newWindow.refreshModuleGui()
}

func LoadProfile(a *SpuWindow) {
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

func loadDialogTrue(a *SpuWindow, reader fyne.URIReadCloser, err error) {
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

	a.Profiles.Exists = true
	a.Profiles.Path = filename
}

func readJson(a *SpuWindow, path string) error {
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
	a.selectModule(a.Modules.MainModule)
	return nil
}
