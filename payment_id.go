package x402

import (
	"crypto/rand"
	"encoding/hex"
)

// GeneratePaymentID generates a random 16-byte payment ID in hex format
func GeneratePaymentID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("x402: failed to generate random payment ID: " + err.Error())
	}
	return "0x" + hex.EncodeToString(b)
}
