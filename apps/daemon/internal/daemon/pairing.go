package daemon

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const pairAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newPairingCode() string {
	var b strings.Builder
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(pairAlphabet))))
		if err != nil {
			panic(err)
		}
		b.WriteByte(pairAlphabet[n.Int64()])
	}
	return b.String()
}
