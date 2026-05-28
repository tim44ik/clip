package storage

import (
	"bytes"
	"clip/errors"
	"clip/models/modules"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type jsonProfile struct{}

func (j *jsonProfile) GetFileType() string {
	return ".json"
}
func (j *jsonProfile) Encode(mods *modules.ClipModules, path string) (error, string) {
	path = strings.TrimSuffix(path, filepath.Ext(path))
	path += ".json"

	file, err := os.Create(path)
	if err != nil {
		return errors.New(errCreatingFile), ""
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	err = encoder.Encode(mods)
	if err != nil {
		return errors.New(errCreatingFile), ""
	}
	return nil, path
}

func (j *jsonProfile) Decode(mods *modules.ClipModules, fileData []byte) error {

	decoder := json.NewDecoder(bytes.NewBuffer(fileData))
	if err := decoder.Decode(mods); err != nil {
		return errors.New(errDecodingFile)
	}

	if mods.MainModule == nil {
		if mods.CurrentLang == "ru" {
			mods.MainModule = &modules.Module{
				Name: "Главная",
			}
		} else {
			mods.MainModule = &modules.Module{
				Name: "Main",
			}
			mods.CurrentLang = "en"
		}

	}

	return nil
}
