package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

func SaveProfile(a *ClipWindow) {
	a.applyModuleChanges()
	switch a.profiles.exists {
	case true:
		makeJson(a, a.profiles.path)
	case false:
		SaveProfileAs(a)
	}
}

func SaveProfileAs(a *ClipWindow) {
	a.applyModuleChanges()

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

			err = makeJson(a, path)
			if err != nil {
				dialog.ShowError(err, a.Window)
				return
			}

			a.profiles.exists = true
			if filepath.Ext(path) != ".json" {
				defer os.Remove(path)
			}

		}, a.Window)
	filesavedialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	filesavedialog.Resize(fyne.NewSize(900, 500))
	filesavedialog.Show()
}

func makeJson(a *ClipWindow, filename string) error {
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	filename += ".json"
	a.profiles.path = filename
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	return encoder.Encode(a.Modules)
}
