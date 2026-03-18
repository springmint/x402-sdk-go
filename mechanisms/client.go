package mechanisms

// ClientMechanism interface is defined in x402 package.
// This package provides implementations:
//
// EVM:   client.Register("eip155:*", &mechanisms.Permit402EvmClientMechanism{Signer: evmSigner})
// Tron:  client.Register("tron:*", &mechanisms.Permit402TronClientMechanism{Signer: tronSigner})
