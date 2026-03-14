package x402

import (
	"context"
)

// FacilitatorClient is the interface for facilitator HTTP client
type FacilitatorClient interface {
	FeeQuote(ctx context.Context, accepts []PaymentRequirements) ([]*FeeQuoteResponse, error)
	Verify(ctx context.Context, payload *PaymentPayload, requirements PaymentRequirements) (*VerifyResponse, error)
	Settle(ctx context.Context, payload *PaymentPayload, requirements PaymentRequirements) (*SettleResponse, error)
	// GetPaymentConfig fetches payment config by apiKey and priceUSD; apiKey is sent via X-API-KEY header
	GetPaymentConfig(ctx context.Context, apiKey string, priceUSD string) ([]PaymentConfigItem, error)
}
