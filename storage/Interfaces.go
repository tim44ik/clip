package storage

import "clip/modules"

type Encoder interface {
	GetFileType() string
	Encode(*modules.ClipModules, string) (error, string)
}

type Decoder interface {
	GetFileType() string
	Decode(*modules.ClipModules, []byte) error
}
