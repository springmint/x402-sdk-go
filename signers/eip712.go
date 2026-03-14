package signers

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EIP712Hash computes the EIP-712 digest for signing
func EIP712Hash(domain, types map[string]any, message map[string]any, primaryType string) ([]byte, error) {
	domainSeparator, err := hashStruct("EIP712Domain", domain, types)
	if err != nil {
		return nil, err
	}
	messageHash, err := hashStruct(primaryType, message, types)
	if err != nil {
		return nil, err
	}

	// keccak256("\x19\x01" ‖ domainSeparator ‖ structHash)
	data := append(common.FromHex("0x1901"), domainSeparator...)
	data = append(data, messageHash...)
	digest := crypto.Keccak256(data)
	return digest, nil
}

func hashStruct(primaryType string, data map[string]any, types map[string]any) ([]byte, error) {
	typeHash := typeHash(primaryType, types)
	encodedValues, err := encodeData(primaryType, data, types)
	if err != nil {
		return nil, err
	}
	return crypto.Keccak256(append(typeHash, encodedValues...)), nil
}

func typeHash(primaryType string, types map[string]any) []byte {
	encoded := encodeType(primaryType, types)
	return crypto.Keccak256([]byte(encoded))
}

func encodeType(primaryType string, types map[string]any) string {
	result := encodeSingleType(primaryType, types)

	deps := findDependencies(primaryType, types, make(map[string]bool))
	delete(deps, primaryType)

	var sortedDeps []string
	for dep := range deps {
		sortedDeps = append(sortedDeps, dep)
	}
	sortStrings(sortedDeps)

	for _, dep := range sortedDeps {
		result += encodeSingleType(dep, types)
	}
	return result
}

func encodeSingleType(typeName string, types map[string]any) string {
	fields, ok := types[typeName].([]map[string]string)
	if !ok {
		return typeName + "()"
	}
	var parts []string
	for _, f := range fields {
		name, _ := f["name"]
		typ, _ := f["type"]
		parts = append(parts, typ+" "+name)
	}
	return typeName + "(" + join(parts, ",") + ")"
}

func findDependencies(typeName string, types map[string]any, visited map[string]bool) map[string]bool {
	if visited[typeName] {
		return visited
	}
	fields, ok := types[typeName].([]map[string]string)
	if !ok {
		return visited
	}
	visited[typeName] = true
	for _, f := range fields {
		typ, _ := f["type"]
		if _, isStruct := types[typ].([]map[string]string); isStruct {
			findDependencies(typ, types, visited)
		}
	}
	return visited
}

func sortStrings(s []string) {
	sort.Strings(s)
}

func encodeData(primaryType string, data map[string]any, types map[string]any) ([]byte, error) {
	fields, ok := types[primaryType].([]map[string]string)
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", primaryType)
	}
	var result []byte
	for _, f := range fields {
		name, _ := f["name"]
		typ, _ := f["type"]
		val, ok := data[name]
		if !ok {
			val = getZeroValue(typ)
		}
		encoded, err := encodeValue(typ, val, types)
		if err != nil {
			return nil, err
		}
		result = append(result, encoded...)
	}
	return result, nil
}

func encodeValue(typ string, val any, types map[string]any) ([]byte, error) {
	switch typ {
	case "address":
		return encodeAddress(val)
	case "uint8", "uint256":
		return encodeUint(val)
	case "bytes16":
		return encodeBytes16(val)
	case "bytes32":
		return encodeBytes32(val)
	case "bool":
		return encodeBool(val)
	case "string":
		return crypto.Keccak256([]byte(fmt.Sprint(val))), nil
	default:
		// Nested struct - hash it
		if fields, ok := types[typ].([]map[string]string); ok && len(fields) > 0 {
			m, ok := val.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map for struct type %s", typ)
			}
			typeH := typeHash(typ, types)
			encoded, err := encodeData(typ, m, types)
			if err != nil {
				return nil, err
			}
			return crypto.Keccak256(append(typeH, encoded...)), nil
		}
		return nil, fmt.Errorf("unsupported type: %s", typ)
	}
}

func encodeAddress(val any) ([]byte, error) {
	var s string
	switch v := val.(type) {
	case string:
		s = v
	case []byte:
		return common.LeftPadBytes(v, 32), nil
	default:
		s = fmt.Sprint(v)
	}
	addr := common.HexToAddress(s)
	return common.LeftPadBytes(addr.Bytes(), 32), nil
}

func encodeUint(val any) ([]byte, error) {
	var n *big.Int
	switch v := val.(type) {
	case *big.Int:
		n = v
	case int:
		n = big.NewInt(int64(v))
	case int64:
		n = big.NewInt(v)
	case uint8:
		n = big.NewInt(int64(v))
	case string:
		n = new(big.Int)
		n.SetString(v, 10)
	default:
		n = big.NewInt(0)
	}
	return common.LeftPadBytes(n.Bytes(), 32), nil
}

func encodeBytes16(val any) ([]byte, error) {
	var b []byte
	switch v := val.(type) {
	case []byte:
		b = v
	case string:
		var err error
		if len(v) >= 2 && v[:2] == "0x" {
			b, err = hex.DecodeString(v[2:])
		} else {
			b, err = hex.DecodeString(v)
		}
		if err != nil {
			return nil, fmt.Errorf("encodeBytes16: invalid hex %q: %w", v, err)
		}
	default:
		b = make([]byte, 16)
	}
	// bytes16: right-pad to 32 bytes for ABI encoding
	if len(b) > 16 {
		b = b[:16]
	}
	return common.RightPadBytes(b, 32), nil
}

func encodeBytes32(val any) ([]byte, error) {
	var b []byte
	switch v := val.(type) {
	case []byte:
		b = v
	case string:
		var err error
		if len(v) >= 2 && v[:2] == "0x" {
			b, err = hex.DecodeString(v[2:])
		} else {
			b, err = hex.DecodeString(v)
		}
		if err != nil {
			return nil, fmt.Errorf("encodeBytes32: invalid hex %q: %w", v, err)
		}
	default:
		b = make([]byte, 32)
	}
	if len(b) > 32 {
		b = b[:32]
	}
	return common.LeftPadBytes(b, 32), nil
}

func encodeBool(val any) ([]byte, error) {
	var n int64
	if b, ok := val.(bool); ok && b {
		n = 1
	}
	return common.LeftPadBytes(big.NewInt(n).Bytes(), 32), nil
}

func getZeroValue(typ string) any {
	switch typ {
	case "address":
		return "0x0000000000000000000000000000000000000000" // keep inline for EIP-712 zero value
	case "uint8", "uint256":
		return big.NewInt(0)
	case "bytes16":
		return make([]byte, 16)
	case "bytes32":
		return make([]byte, 32)
	case "bool":
		return false
	default:
		return nil
	}
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// Helper to convert int64 to bytes for validAfter/validBefore
func int64ToBytes(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return common.LeftPadBytes(b, 32)
}
