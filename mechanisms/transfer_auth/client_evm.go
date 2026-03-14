package transfer_auth

import (
	"context"
	"strconv"

	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/signers"
	"github.com/springmint/x402-sdk-go/tokens"
)

// TransferAuthEvmClientMechanism implements transfer_auth (TransferWithAuthorization) for EVM
type TransferAuthEvmClientMechanism struct {
	Signer  signers.ClientSigner
	Adapter ChainAdapter
}

// NewTransferAuthEvmClientMechanism creates an EVM transfer_auth client mechanism
func NewTransferAuthEvmClientMechanism(signer signers.ClientSigner) *TransferAuthEvmClientMechanism {
	return &TransferAuthEvmClientMechanism{
		Signer:  signer,
		Adapter: EvmChainAdapter{},
	}
}

// Scheme returns "transfer_auth"
func (m *TransferAuthEvmClientMechanism) Scheme() string {
	return SchemeTransferAuth
}

// CreatePaymentPayload creates signed PaymentPayload with TransferAuthorization in extensions
func (m *TransferAuthEvmClientMechanism) CreatePaymentPayload(ctx context.Context, requirements x402.PaymentRequirements, resource string, extensions map[string]any) (*x402.PaymentPayload, error) {
	adapter := m.Adapter
	fromAddr := adapter.ToSigningAddress(m.Signer.GetAddress())
	toAddr := adapter.ToSigningAddress(requirements.PayTo)
	value := requirements.Amount
	tokenAddress := requirements.Asset

	tokenInfo := tokens.FindByAddress(requirements.Network, tokenAddress)
	tokenName := "Unknown Token"
	tokenVersion := "1"
	if tokenInfo != nil {
		tokenName = tokenInfo.Name
		tokenVersion = tokenInfo.Version
		if tokenVersion == "" {
			tokenVersion = "1"
		}
	}

	validAfter, validBefore := CreateValidityWindow(DefaultValiditySeconds)
	nonce := CreateNonce()

	auth := TransferAuthorization{
		From:        fromAddr,
		To:          toAddr,
		Value:       value,
		ValidAfter:  strconv.FormatInt(validAfter, 10),
		ValidBefore: strconv.FormatInt(validBefore, 10),
		Nonce:       nonce,
	}

	chainID, err := adapter.ParseChainID(requirements.Network)
	if err != nil {
		return nil, err
	}
	domain := map[string]any{
		"name":              tokenName,
		"version":           tokenVersion,
		"chainId":           chainID,
		"verifyingContract": adapter.ToSigningAddress(tokenAddress),
	}
	typesMap := typesToMapAny(TransferAuthEIP712Types())
	nonceBytes := HexToBytes(nonce)
	msgMap := map[string]any{
		"from":        auth.From,
		"to":          auth.To,
		"value":       auth.Value,
		"validAfter":  auth.ValidAfter,
		"validBefore": auth.ValidBefore,
		"nonce":       nonceBytes,
	}

	signature, err := m.Signer.SignTypedData(ctx, domain, typesMap, msgMap, TransferAuthPrimaryType)
	if err != nil {
		return nil, err
	}

	authDict := map[string]any{
		"from":        auth.From,
		"to":          auth.To,
		"value":       auth.Value,
		"validAfter":  auth.ValidAfter,
		"validBefore": auth.ValidBefore,
		"nonce":       auth.Nonce,
	}
	return &x402.PaymentPayload{
		X402Version: 2,
		Resource:    &x402.ResourceInfo{URL: resource},
		Accepted:    requirements,
		Payload: x402.PaymentPayloadData{
			Signature:     signature,
			Permit402: nil,
		},
		Extensions: map[string]any{
			"transferAuthorization": authDict,
		},
	}, nil
}

func typesToMapAny(t map[string][]map[string]string) map[string]any {
	out := make(map[string]any)
	for k, v := range t {
		out[k] = v
	}
	return out
}

