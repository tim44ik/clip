package encrypter

import "clip/errors"

const (
	errReadingProfile   errors.Code = "profile_reading_error"
	errInvalidData      errors.Code = "invalid_data_error"
	errGeneratingSalt   errors.Code = "salt_generation_error"
	errGeneratingKey    errors.Code = "key_generation_error"
	errGeneratingCipher errors.Code = "cipher_generation_error"
	errGeneratingGCM    errors.Code = "gcm_generation_error"
	errGeneratingNonce  errors.Code = "nonce_generation_error"
	errWritingToFile    errors.Code = "write_error"
)
