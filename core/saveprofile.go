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

func SaveProfile(a *SpuWindow) {
	a.applyModuleChanges()
	switch a.Profiles.Exists {
	case true:
		makeJson(a, a.Profiles.Path)
	case false:
		SaveProfileAs(a)
	}
}

func SaveProfileAs(a *SpuWindow) {
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

			a.Profiles.Exists = true
			if filepath.Ext(path) != ".json" {
				defer os.Remove(path)
			}

		}, a.Window)
	filesavedialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	filesavedialog.Resize(fyne.NewSize(900, 500))
	filesavedialog.Show()
}

func makeJson(a *SpuWindow, filename string) error {
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
