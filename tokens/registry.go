package tokens

import (
	"strconv"
	"strings"

	"github.com/springmint/x402-sdk-go/tron"
)

// TokenInfo holds token metadata
type TokenInfo struct {
	Address  string
	Decimals int
	Name     string
	Symbol   string
	Version  string
	// SupportsTransferWithAuthorization: true if token implements ERC-3009 transferWithAuthorization.
	// If false, only permit402 scheme can be used (Permit402 contract + approve/transferFrom).
	SupportsTransferWithAuthorization bool
}

var registry = map[string]map[string]*TokenInfo{
	"eip155:1": {
		"USDC": {Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Decimals: 6, Name: "USD Coin", Symbol: "USDC", Version: "1", SupportsTransferWithAuthorization: true},
		"USDT": {Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Decimals: 6, Name: "Tether USD", Symbol: "USDT", Version: "1", SupportsTransferWithAuthorization: false},
	},
	"eip155:56": {
		"USDC": {Address: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", Decimals: 18, Name: "USD Coin", Symbol: "USDC", Version: "1", SupportsTransferWithAuthorization: true},
		"USDT": {Address: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18, Name: "Tether USD", Symbol: "USDT", Version: "1", SupportsTransferWithAuthorization: false},
	},
	"eip155:97": {
		"USDT": {Address: "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", Decimals: 18, Name: "Tether USD", Symbol: "USDT", Version: "1", SupportsTransferWithAuthorization: false},
		"USDC": {Address: "0x64544969ed7EBf5f083679233325356EbE738930", Decimals: 18, Name: "USDC Token", Symbol: "USDC", Version: "1", SupportsTransferWithAuthorization: true},
	},
	"eip155:8453": {
		// Base native USDC (FiatTokenV2): EIP-712 domain uses version "2"
		"USDC": {Address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", Decimals: 6, Name: "USD Coin", Symbol: "USDC", Version: "2", SupportsTransferWithAuthorization: true},
		"USDT": {Address: "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2", Decimals: 6, Name: "Tether USD", Symbol: "USDT", Version: "1", SupportsTransferWithAuthorization: false},
	},
	// TRON Networks
	"tron:mainnet": {
		"USDT": {Address: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", Decimals: 6, Name: "Tether USD", Symbol: "USDT", Version: "1", SupportsTransferWithAuthorization: false},
		"USDD": {Address: "TXDk8mbtRbXeYuMNS83CfKPaYYT8XWv9Hz", Decimals: 18, Name: "Decentralized USD", Symbol: "USDD", Version: "1", SupportsTransferWithAuthorization: false},
	},
	"tron:shasta": {
		"USDT": {Address: "TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs", Decimals: 6, Name: "Tether USD", Symbol: "USDT", Version: "1", SupportsTransferWithAuthorization: false},
	},
	"tron:nile": {
		"USDT": {Address: "TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf", Decimals: 6, Name: "Tether USD", Symbol: "USDT", Version: "1", SupportsTransferWithAuthorization: false},
		"USDD": {Address: "TGjgvdTWWrybVLaVeFqSyVqJQWjxqRYbaK", Decimals: 18, Name: "Decentralized USD", Symbol: "USDD", Version: "1", SupportsTransferWithAuthorization: false},
	},
}

// FindByAddress finds token by address
func FindByAddress(network, address string) *TokenInfo {
	tokens := registry[network]
	if tokens == nil {
		return nil
	}
	addrToMatch := address
	if strings.HasPrefix(network, "tron:") && len(address) >= 2 && address[:2] == "0x" {
		// Tron registry uses base58; convert hex to base58 for lookup
		if base58, err := tron.HexToBase58(address); err == nil {
			addrToMatch = base58
		}
	}
	for _, t := range tokens {
		if strings.HasPrefix(network, "tron:") {
			if t.Address == addrToMatch {
				return t
			}
		} else {
			if strings.ToLower(t.Address) == strings.ToLower(addrToMatch) {
				return t
			}
		}
	}
	return nil
}

// GetToken returns token by symbol
func GetToken(network, symbol string) *TokenInfo {
	tokens := registry[network]
	if tokens == nil {
		return nil
	}
	return tokens[strings.ToUpper(symbol)]
}

// SupportsTransferAuth returns true if the token supports the "transfer_auth" scheme (ERC-3009 transferWithAuthorization).
// Tokens with false should only use "permit402" (ERC-20 approve + Permit402 contract).
func (t *TokenInfo) SupportsTransferAuth() bool {
	return t != nil && t.SupportsTransferWithAuthorization
}

// ParsePrice parses "100 USDC" style price
func ParsePrice(price, network string) (amount string, asset string, err error) {
	parts := strings.Fields(strings.TrimSpace(price))
	if len(parts) != 2 {
		return "", "", errInvalidPrice
	}
	amountStr, symbol := parts[0], parts[1]
	amt, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return "", "", err
	}
	token := GetToken(network, symbol)
	if token == nil {
		return "", "", errUnknownToken
	}
	amountSmallest := int64(amt * float64(pow10(token.Decimals)))
	return strconv.FormatInt(amountSmallest, 10), token.Address, nil
}

func pow10(n int) int64 {
	r := int64(1)
	for i := 0; i < n; i++ {
		r *= 10
	}
	return r
}

var errInvalidPrice = &parseError{"invalid price format"}
var errUnknownToken = &parseError{"unknown token"}

type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }
