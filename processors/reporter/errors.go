package reporter

import "clip/errors"

const (
	errWritingToFile     errors.Code = "report_file_writing_error"
	errWrongReportFormat errors.Code = "report_file_type_error"
)
