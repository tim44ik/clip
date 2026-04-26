package storage

import "clip/errors"

const (
	errCreatingFile errors.Code = "error_saving_profile"
	errEncodingFile errors.Code = "encoding_error"
	errDecodingFile errors.Code = "decoding_error"
)
