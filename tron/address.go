package tron

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
)

// Tron address: 21 bytes = 0x41 + 20 bytes, base58check encoded.
// For EIP-712/TIP-712 we use the 20-byte part as address (0x + 40 hex chars).

// Base58 alphabet used by Tron (same as Bitcoin)
const b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// TronAddressPrefix is the byte prefix for Tron addresses
const TronAddressPrefix = 0x41

// Base58ToHex decodes a Tron base58check address and returns the 20-byte address as hex (0x + 40 chars).
// Input: "TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf"
// Output: "0x" + hex of 20 bytes (address part, without 0x41 prefix)
func Base58ToHex(base58 string) (string, error) {
	decoded, err := base58Decode(base58)
	if err != nil {
		return "", err
	}
	if len(decoded) != 25 {
		return "", errors.New("invalid Tron address: expected 25 bytes after decode")
	}
	payload := decoded[:21]
	checksum := decoded[21:]
	expectedChecksum := doubleSha256(payload)[:4]
	for i := 0; i < 4; i++ {
		if checksum[i] != expectedChecksum[i] {
			return "", errors.New("invalid Tron address: checksum mismatch")
		}
	}
	if payload[0] != TronAddressPrefix {
		return "", errors.New("invalid Tron address: wrong prefix")
	}
	addr20 := payload[1:]
	return "0x" + hex.EncodeToString(addr20), nil
}

// ToHex converts address to hex (0x + 40 chars). Accepts either base58 (T...) or hex (0x...).
func ToHex(addr string) (string, error) {
	if addr == "" {
		return "", errors.New("empty address")
	}
	if len(addr) >= 2 && addr[:2] == "0x" {
		// Already hex - validate and return
		h := strings.TrimPrefix(addr, "0x")
		if len(h) != 40 {
			return "", errors.New("invalid hex address: expected 40 hex chars")
		}
		return "0x" + h, nil
	}
	return Base58ToHex(addr)
}

// HexToBase58 encodes a 20-byte hex address (with or without 0x) to Tron base58check.
func HexToBase58(hexAddr string) (string, error) {
	if len(hexAddr) >= 2 && hexAddr[:2] == "0x" {
		hexAddr = hexAddr[2:]
	}
	if len(hexAddr) != 40 {
		return "", errors.New("invalid hex address: expected 40 hex chars")
	}
	addr20, err := hex.DecodeString(hexAddr)
	if err != nil {
		return "", err
	}
	payload := make([]byte, 21)
	payload[0] = TronAddressPrefix
	copy(payload[1:], addr20)
	checksum := doubleSha256(payload)[:4]
	full := append(payload, checksum...)
	return base58Encode(full), nil
}

func doubleSha256(b []byte) []byte {
	h := sha256.Sum256(b)
	h2 := sha256.Sum256(h[:])
	return h2[:]
}

func base58Encode(b []byte) string {
	var (
		zeroCount int
		size      = len(b)*138/100 + 1
		buf       = make([]byte, size)
	)
	for _, c := range b {
		if c != 0 {
			break
		}
		zeroCount++
	}
	for _, v := range b {
		carry := uint32(v)
		for j := size - 1; j >= 0; j-- {
			carry += 256 * uint32(buf[j])
			buf[j] = byte(carry % 58)
			carry /= 58
		}
	}
	i := 0
	for i < size && buf[i] == 0 {
		i++
	}
	out := make([]byte, zeroCount+size-i)
	for j := 0; j < zeroCount; j++ {
		out[j] = '1'
	}
	for k := i; k < size; k++ {
		out[zeroCount+k-i] = b58Alphabet[buf[k]]
	}
	return string(out)
}

func base58Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return nil, errors.New("empty base58 string")
	}
	var b58Table [256]int
	for i := range b58Table {
		b58Table[i] = -1
	}
	for i, c := range b58Alphabet {
		b58Table[c] = i
	}
	n := big.NewInt(0)
	for _, c := range s {
		if c > 127 {
			return nil, errors.New("invalid base58 character")
		}
		val := b58Table[c]
		if val < 0 {
			return nil, errors.New("invalid base58 character")
		}
		n.Mul(n, big.NewInt(58))
		n.Add(n, big.NewInt(int64(val)))
	}
	raw := n.Bytes()
	zcount := 0
	for _, c := range s {
		if c != '1' {
			break
		}
		zcount++
	}
	out := make([]byte, zcount+len(raw))
	for i := 0; i < zcount; i++ {
		out[i] = 0
	}
	copy(out[zcount:], raw)
	return out, nil
}

