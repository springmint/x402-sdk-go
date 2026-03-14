package transfer_auth

import (
	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/tokens"
)

// TransferAuthEvmServerMechanism implements transfer_auth server for EVM
type TransferAuthEvmServerMechanism struct {
	Adapter ChainAdapter
}

// NewTransferAuthEvmServerMechanism creates an EVM transfer_auth server mechanism
func NewTransferAuthEvmServerMechanism() *TransferAuthEvmServerMechanism {
	return &TransferAuthEvmServerMechanism{Adapter: EvmChainAdapter{}}
}

// Scheme returns "transfer_auth"
func (m *TransferAuthEvmServerMechanism) Scheme() string {
	return SchemeTransferAuth
}

// ParsePrice parses "100 USDC" format
func (m *TransferAuthEvmServerMechanism) ParsePrice(price, network string) (amount, asset string, err error) {
	return tokens.ParsePrice(price, network)
}

// ValidatePaymentRequirements validates requirements
func (m *TransferAuthEvmServerMechanism) ValidatePaymentRequirements(requirements x402.PaymentRequirements) bool {
	adapter := m.Adapter
	if !adapter.ValidateNetwork(requirements.Network) {
		return false
	}
	if !adapter.ValidateAddress(requirements.Asset) {
		return false
	}
	if !adapter.ValidateAddress(requirements.PayTo) {
		return false
	}
	if requirements.Amount == "" || requirements.Amount == "0" {
		return false
	}
	return true
}
