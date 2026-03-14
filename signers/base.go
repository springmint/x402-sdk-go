package signers

import (
	"context"
	"math/big"
)

// ClientSigner is the interface for client signers
type ClientSigner interface {
	GetAddress() string
	SignMessage(ctx context.Context, message []byte) (string, error)
	SignTypedData(ctx context.Context, domain map[string]any, types map[string]any, message map[string]any, primaryType string) (string, error)
	CheckBalance(ctx context.Context, token, network string) (*big.Int, error)
	CheckAllowance(ctx context.Context, token string, amount *big.Int, network string) (*big.Int, error)
	EnsureAllowance(ctx context.Context, token string, amount *big.Int, network string, mode string) (bool, error)
}
