package reporter

import (
	"clip/modules"
	outputprocessor "clip/outputProcessor"
)

type Reporter interface {
	CreateReport(outputprocessor.DB, []*modules.Module, string, chan<- float64, chan<- error)
	GetFileType() string
}
