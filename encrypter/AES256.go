package encrypter

import (
	"bytes"
	"clip/errors"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"os"

	"golang.org/x/crypto/scrypt"
)

type aes256 struct{}

func (a *aes256) Encrypt(filename, password string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return errors.New(errReadingProfile)
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return errors.New(errGeneratingSalt)
	}

	key, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return errors.New(errGeneratingKey)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(errGeneratingCipher)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return errors.New(errGeneratingGCM)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	payload := append(salt, nonce...)
	payload = append(payload, ciphertext...)

	var buf bytes.Buffer

	buf.Write([]byte("CFG1"))
	buf.WriteByte(1)
	buf.WriteByte(0x01)
	buf.WriteByte(1)
	buf.Write(payload)

	err = os.WriteFile(filename, buf.Bytes(), 0600)
	if err != nil {
		return errors.New(errWritingToFile)
	}
	return nil
}

func (a *aes256) Decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < 16 {
		return nil, errors.New(errInvalidData)
	}

	salt := data[:16]

	key, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return nil, errors.New(errGeneratingKey)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New(errGeneratingCipher)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New(errGeneratingGCM)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < 16+nonceSize {
		return nil, errors.New(errInvalidData)
	}

	nonce := data[16 : 16+nonceSize]
	ciphertext := data[16+nonceSize:]

	bytes, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New(errReadingProfile)
	}

	return bytes, nil
}
