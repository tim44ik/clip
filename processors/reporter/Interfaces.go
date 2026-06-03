package reporter

import (
	"clip/errors"
)

type Reporter interface {
	CreateReport(string, []*ReportContent, chan<- error)
	GetFileType() string
}

type Report struct {
	Reporter Reporter
	Content  []*ReportContent
}

type ReportContent struct {
	Mname string
	Body  string
}

func NewReport() *Report {
	return &Report{Content: make([]*ReportContent, 0)}
}

func (r *Report) NewReportContent(mName string) *ReportContent {
	return &ReportContent{Mname: mName}
}

func (r *Report) NewReporter(rtype string) (Reporter, error) {
	switch rtype {
	case ".pdf":
		return &pdf{}, nil
	default:
		return nil, errors.New(errWrongReportFormat)
	}
}
