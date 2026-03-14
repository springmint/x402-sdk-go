package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GeneratePaymentID generates a random 16-byte payment ID in hex format
func GeneratePaymentID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "0x" + hex.EncodeToString(b)
}
