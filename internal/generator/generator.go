package generator

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	alphabet   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	codeLength = 10
)

func Generate() (string, error) {
	code := make([]byte, codeLength)

	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate short code: %w", err)
		}

		code[i] = alphabet[n.Int64()]
	}

	return string(code), nil
}
