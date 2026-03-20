package storage

import (
	"bytes"
	"clip/modules"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Json struct {
}

func NewJson() *Json {
	return &Json{}
}

func (j *Json) GetFileType() string {
	return ".json"
}
func (j *Json) Encode(mods *modules.ClipModules, path string) (error, string) {
	path = strings.TrimSuffix(path, filepath.Ext(path))
	path += ".json"

	file, err := os.Create(path)
	if err != nil {
		return err, ""
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	return encoder.Encode(mods), path
}

func (j *Json) Decode(mods *modules.ClipModules, fileData []byte) error {

	decoder := json.NewDecoder(bytes.NewBuffer(fileData))
	if e := decoder.Decode(mods); e != nil {
		return e
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
