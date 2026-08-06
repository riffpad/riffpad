package hub

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"strings"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newCodeN(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeAlphabet))))
		if err != nil {
			panic(err)
		}
		b.WriteByte(codeAlphabet[n.Int64()])
	}
	return b.String()
}

func newCode() string {
	return newCodeN(6)
}

func newDeviceCode() string {
	return newCodeN(8)
}

func newSecret() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
