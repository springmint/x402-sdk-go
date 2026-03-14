package signers

import (
	"context"
)

// FacilitatorSigner is the interface for facilitator signers
type FacilitatorSigner interface {
	GetAddress() string
	VerifyTypedData(ctx context.Context, address string, domain, types, message map[string]any, signature string, primaryType string) (bool, error)
	WriteContract(ctx context.Context, contractAddress, abiJSON, method string, args []any, network string) (string, error)
	WaitForTransactionReceipt(ctx context.Context, txHash, network string, timeout int) (map[string]any, error)
}
