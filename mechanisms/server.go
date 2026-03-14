package mechanisms

import (
	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/tokens"
)

// Permit402EvmServerMechanism implements permit402 server for EVM
type Permit402EvmServerMechanism struct{}

// Scheme returns "permit402"
func (m *Permit402EvmServerMechanism) Scheme() string {
	return "permit402"
}

// ParsePrice parses "100 USDC" format
func (m *Permit402EvmServerMechanism) ParsePrice(price, network string) (amount, asset string, err error) {
	return tokens.ParsePrice(price, network)
}

// ValidatePaymentRequirements validates requirements
func (m *Permit402EvmServerMechanism) ValidatePaymentRequirements(requirements x402.PaymentRequirements) bool {
	if requirements.Network == "" || requirements.Asset == "" || requirements.PayTo == "" {
		return false
	}
	if requirements.Amount == "" || requirements.Amount == "0" {
		return false
	}
	return true
}
