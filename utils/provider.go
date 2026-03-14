package utils

import (
	"strings"

	x402 "github.com/springmint/x402-sdk-go"
)

// ResolveProviderURI resolves network to RPC provider URI
func ResolveProviderURI(network string) string {
	if strings.HasPrefix(network, "http://") || strings.HasPrefix(network, "https://") ||
		strings.HasPrefix(network, "ws://") || strings.HasPrefix(network, "wss://") {
		return network
	}
	return x402.GetRPCURL(network)
}
