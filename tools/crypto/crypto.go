package cryptoTool

type ICrypto interface {
	GenerateFromPassword(password string) (encodedHash string, err error)
	ComparePasswordAndHash(password, encodedHash string) (match bool, err error)
}
