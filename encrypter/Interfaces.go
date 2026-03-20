package encrypter

const (
	CipherAES256SCRYPT = 1
)

type Encrypter interface {
	Encrypt(string, string) error
}

type Decrypter interface {
	Decrypt([]byte, string) ([]byte, error)
}
