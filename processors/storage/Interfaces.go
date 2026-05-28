package storage

import "clip/models/modules"

type Encoder interface {
	GetFileType() string
	Encode(*modules.ClipModules, string) (error, string)
}

type Decoder interface {
	GetFileType() string
	Decode(*modules.ClipModules, []byte) error
}

func NewEncoder(ptype string) Encoder {
	switch ptype {
	case ".json":
		return &jsonProfile{}
	default:
		return nil
	}
}

func NewDecoder(ptype string) Decoder {
	switch ptype {
	case ".json":
		return &jsonProfile{}
	default:
		return nil
	}
}
