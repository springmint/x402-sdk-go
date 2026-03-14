package x402

import (
	"encoding/base64"
	"encoding/json"
)

// EncodeBase64 encodes data to base64
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 to bytes
func DecodeBase64(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// EncodePaymentPayload encodes payment payload to base64 for HTTP header
func EncodePaymentPayload(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return EncodeBase64(data), nil
}

// DecodePaymentPayload decodes payment payload from base64 HTTP header
func DecodePaymentPayload[T any](encoded string) (*T, error) {
	bytes, err := DecodeBase64(encoded)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(bytes, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BytesToHex converts bytes to hex string
func BytesToHex(data []byte, prefix bool) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, len(data)*2)
	for i, v := range data {
		b[i*2] = hexChars[v>>4]
		b[i*2+1] = hexChars[v&0x0f]
	}
	if prefix {
		return "0x" + string(b)
	}
	return string(b)
}

// HexToBytes converts hex string to bytes
func HexToBytes(hexStr string) ([]byte, error) {
	if len(hexStr) >= 2 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	bytes := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		var b byte
		for _, c := range hexStr[i : i+2] {
			b <<= 4
			switch {
			case c >= '0' && c <= '9':
				b |= byte(c - '0')
			case c >= 'a' && c <= 'f':
				b |= byte(c - 'a' + 10)
			case c >= 'A' && c <= 'F':
				b |= byte(c - 'A' + 10)
			default:
				return nil, &X402Error{Msg: "invalid hex character"}
			}
		}
		bytes[i/2] = b
	}
	return bytes, nil
}
