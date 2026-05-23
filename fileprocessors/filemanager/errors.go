package filemanager

import "clip/errors"

const (
	errSavingProfile       errors.Code = "saving_profile_error"
	errReadingProfile      errors.Code = "profile_reading_error"
	errLoadingProfile      errors.Code = "loading_profile_error"
	errPasswordNotProvided errors.Code = "password_not_provided"
	errInvalidData         errors.Code = "invalid_data_error"
	errUnknownCipher       errors.Code = "unknown_cipher"
	errReadingFile         errors.Code = "file_read_error"
	errOpeningFolder       errors.Code = "open_folder_error"
	errListingFiles        errors.Code = "listing_files_error"
	errReportType          errors.Code = "report_file_type_error"
)
