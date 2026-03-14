package transfer_auth

import (
	"strconv"
	"strings"

	x402 "github.com/springmint/x402-sdk-go"
)

// ChainAdapter encapsulates chain-specific behaviour for transfer_auth
type ChainAdapter interface {
	ParseChainID(network string) (int64, error)
	ValidateNetwork(network string) bool
	ValidateAddress(address string) bool
	NormalizeAddress(address string) string
	ToSigningAddress(address string) string
}

// EvmChainAdapter is the adapter for EVM (eip155:<chainId>)
type EvmChainAdapter struct{}

func (EvmChainAdapter) ParseChainID(network string) (int64, error) {
	if !strings.HasPrefix(network, "eip155:") {
		return 0, x402.NewUnsupportedNetworkError("not an EVM network: " + network)
	}
	parts := strings.SplitN(network, ":", 2)
	if len(parts) != 2 {
		return 0, x402.NewUnsupportedNetworkError("invalid EVM network: " + network)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (EvmChainAdapter) ValidateNetwork(network string) bool {
	return strings.HasPrefix(network, "eip155:")
}

func (EvmChainAdapter) ValidateAddress(address string) bool {
	return strings.HasPrefix(address, "0x") && len(address) == 42
}

func (EvmChainAdapter) NormalizeAddress(address string) string {
	return strings.ToLower(address)
}

func (EvmChainAdapter) ToSigningAddress(address string) string {
	return address
}
