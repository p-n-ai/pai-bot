package group

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const joinCodeCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
const joinCodeLength = 6

func GenerateJoinCode() (string, error) {
	b := make([]byte, joinCodeLength)
	max := big.NewInt(int64(len(joinCodeCharset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = joinCodeCharset[n.Int64()]
	}
	return string(b), nil
}

func NormalizeJoinCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
