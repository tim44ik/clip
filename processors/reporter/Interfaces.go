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
	Rtype    string
	Content  []*ReportContent
}

type ReportContent struct {
	Mname string
	Body  string
}

func NewReport(rtype string) *Report {
	return &Report{Rtype: rtype, Content: make([]*ReportContent, 0)}
}

func (r *Report) NewReportContent(mName string) *ReportContent {
	return &ReportContent{Mname: mName}
}

func (r *Report) NewReporter() error {
	switch r.Rtype {
	case ".pdf":
		r.Reporter = &pdf{}
		return nil
	default:
		return errors.New(errWrongReportFormat)
	}
}
