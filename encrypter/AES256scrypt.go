package encrypter

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/scrypt"
)

type AES256SCRYPT struct{}

func NewAES256SCRYPT() *AES256SCRYPT {
	return &AES256SCRYPT{}
}

func (a *AES256SCRYPT) Encrypt(filename, password string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}

	key, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
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

	return os.WriteFile(filename, buf.Bytes(), 0600)
}

func (a *AES256SCRYPT) Decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("invalid data")
	}

	salt := data[:16]

	key, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < 16+nonceSize {
		return nil, errors.New("invalid data")
	}

	nonce := data[16 : 16+nonceSize]
	ciphertext := data[16+nonceSize:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}
