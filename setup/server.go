package setup

import (
	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/mechanisms"
	"github.com/springmint/x402-sdk-go/mechanisms/transfer_auth"
)

// RegisterEVMChains registers transfer_auth and permit402 for Base, Ethereum, and BSC.
// Call this after NewX402Server() to support eip155:1, eip155:56, eip155:97, eip155:8453.
func RegisterEVMChains(s *x402.X402Server) *x402.X402Server {
	for _, chain := range []string{x402.BaseMainnet, x402.EVMMainnet, x402.BSCMainnet, x402.BSCTestnet} {
		s.Register(chain, transfer_auth.NewTransferAuthEvmServerMechanism())
		s.Register(chain, &mechanisms.Permit402EvmServerMechanism{})
	}
	return s
}

// RegisterTronChains registers permit402 for Tron mainnet, Shasta, and Nile.
// Pass a TronFacilitatorSigner when configuring the facilitator for Settle.
func RegisterTronChains(s *x402.X402Server) *x402.X402Server {
	for _, chain := range []string{x402.TronMainnet, x402.TronShasta, x402.TronNile} {
		s.Register(chain, &mechanisms.Permit402EvmServerMechanism{})
	}
	return s
}
