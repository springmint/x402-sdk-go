package transfer_auth

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

const (
	SchemeTransferAuth      = "transfer_auth"
	DefaultValiditySeconds  = 3600
	ValidityBufferSeconds   = 30
)

// TransferAuthorization is the EIP-3009 TransferWithAuthorization parameters
type TransferAuthorization struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	ValidAfter  string `json:"validAfter"`
	ValidBefore string `json:"validBefore"`
	Nonce       string `json:"nonce"` // 32-byte hex (0x...)
}

// TransferAuthEIP712Types returns EIP-712 type definitions for TransferWithAuthorization.
// Domain for transfer_auth uses name, version, chainId, verifyingContract.
func TransferAuthEIP712Types() map[string][]map[string]string {
	return map[string][]map[string]string{
		"EIP712Domain": {
			{"name": "name", "type": "string"},
			{"name": "version", "type": "string"},
			{"name": "chainId", "type": "uint256"},
			{"name": "verifyingContract", "type": "address"},
		},
		"TransferWithAuthorization": {
			{"name": "from", "type": "address"},
			{"name": "to", "type": "address"},
			{"name": "value", "type": "uint256"},
			{"name": "validAfter", "type": "uint256"},
			{"name": "validBefore", "type": "uint256"},
			{"name": "nonce", "type": "bytes32"},
		},
	}
}

// TransferAuthPrimaryType is the EIP-712 primary type
const TransferAuthPrimaryType = "TransferWithAuthorization"

// CreateNonce generates a random 32-byte nonce (0x-prefixed hex)
func CreateNonce() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("x402: failed to generate random nonce: " + err.Error())
	}
	return "0x" + hex.EncodeToString(b)
}

// CreateValidityWindow returns (validAfter, validBefore) with clock skew buffer
func CreateValidityWindow(duration int) (int64, int64) {
	if duration <= 0 {
		duration = DefaultValiditySeconds
	}
	now := time.Now().Unix()
	return now - ValidityBufferSeconds, now + int64(duration)
}

// HexToBytes decodes 0x-prefixed hex string to bytes.
// Returns nil if the hex string is invalid.
func HexToBytes(s string) []byte {
	s = strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return b
}
