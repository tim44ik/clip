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

func NewEncrypter(etype string) Encrypter {
	switch etype {
	case "AES256":
		return &aes256{}
	default:
		return nil
	}
}

func NewDecrypter(etype int) Decrypter {
	switch etype {
	case 1:
		return &aes256{}
	default:
		return nil
	}
}
