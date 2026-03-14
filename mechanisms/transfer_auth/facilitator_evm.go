package transfer_auth

import (
	"context"
	"math/big"
	"strconv"
	"time"

	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/signers"
	"github.com/springmint/x402-sdk-go/tokens"

	"github.com/ethereum/go-ethereum/common"
)

// TransferAuthEvmFacilitatorMechanism implements transfer_auth facilitator for EVM.
// Fee is always 0 (transfer_auth does not support fee from payment).
type TransferAuthEvmFacilitatorMechanism struct {
	Signer        signers.FacilitatorSigner
	Adapter       ChainAdapter
	AllowedTokens map[string]struct{} // normalized address set, nil = allow all
}

// NewTransferAuthEvmFacilitatorMechanism creates an EVM transfer_auth facilitator mechanism
func NewTransferAuthEvmFacilitatorMechanism(signer signers.FacilitatorSigner, allowedTokens []string) *TransferAuthEvmFacilitatorMechanism {
	m := &TransferAuthEvmFacilitatorMechanism{
		Signer:  signer,
		Adapter: EvmChainAdapter{},
	}
	if len(allowedTokens) > 0 {
		m.AllowedTokens = make(map[string]struct{})
		for _, t := range allowedTokens {
			m.AllowedTokens[m.Adapter.NormalizeAddress(t)] = struct{}{}
		}
	}
	return m
}

// Scheme returns "transfer_auth"
func (m *TransferAuthEvmFacilitatorMechanism) Scheme() string {
	return SchemeTransferAuth
}

// FeeQuote returns fee=0 for transfer_auth
func (m *TransferAuthEvmFacilitatorMechanism) FeeQuote(ctx context.Context, accept x402.PaymentRequirements) (*x402.FeeQuoteResponse, error) {
	return &x402.FeeQuoteResponse{
		Fee: x402.FeeInfo{
			FeeTo:     m.Signer.GetAddress(),
			FeeAmount: "0",
		},
		Pricing:   "flat",
		Scheme:    accept.Scheme,
		Network:   accept.Network,
		Asset:     accept.Asset,
		ExpiresAt: intPtr64(time.Now().Unix() + 300),
	}, nil
}

// Verify validates authorization and EIP-712 signature
func (m *TransferAuthEvmFacilitatorMechanism) Verify(ctx context.Context, payload *x402.PaymentPayload, requirements x402.PaymentRequirements) (*x402.VerifyResponse, error) {
	auth := m.extractAuthorization(payload)
	if auth == nil {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: "missing_transfer_authorization"}, nil
	}
	if err := m.validateAuthorization(auth, requirements); err != "" {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: err}, nil
	}
	ok, err := m.verifySignature(ctx, auth, payload.Payload.Signature, requirements)
	if err != nil || !ok {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: "invalid_signature"}, nil
	}
	return &x402.VerifyResponse{IsValid: true}, nil
}

// Settle calls transferWithAuthorization on the token contract
func (m *TransferAuthEvmFacilitatorMechanism) Settle(ctx context.Context, payload *x402.PaymentPayload, requirements x402.PaymentRequirements) (*x402.SettleResponse, error) {
	resp, err := m.Verify(ctx, payload, requirements)
	if err != nil || !resp.IsValid {
		return &x402.SettleResponse{
			Success:     false,
			ErrorReason: resp.InvalidReason,
			Network:     requirements.Network,
		}, nil
	}
	auth := m.extractAuthorization(payload)
	sig := payload.Payload.Signature
	sigBytes := HexToBytes(sig)
	if len(sigBytes) != 65 {
		return &x402.SettleResponse{
			Success:     false,
			ErrorReason: "invalid_signature_length",
			Network:     requirements.Network,
		}, nil
	}
	var r32, s32 [32]byte
	copy(r32[:], sigBytes[:32])
	copy(s32[:], sigBytes[32:64])
	v := uint8(sigBytes[64])
	if v < 27 {
		v += 27
	}
	fromAddr := common.HexToAddress(auth.From)
	toAddr := common.HexToAddress(auth.To)
	nonceBytes := HexToBytes(auth.Nonce)
	var nonce32 [32]byte
	copy(nonce32[:], nonceBytes)
	value := new(big.Int)
	value.SetString(auth.Value, 10)
	validAfter := new(big.Int)
	validAfter.SetString(auth.ValidAfter, 10)
	validBefore := new(big.Int)
	validBefore.SetString(auth.ValidBefore, 10)

	args := []any{
		fromAddr,
		toAddr,
		value,
		validAfter,
		validBefore,
		nonce32,
		v,
		r32,
		s32,
	}
	txHash, err := m.Signer.WriteContract(ctx, requirements.Asset, TransferWithAuthorizationABI, "transferWithAuthorization", args, requirements.Network)
	if err != nil || txHash == "" {
		return &x402.SettleResponse{
			Success:     false,
			ErrorReason: "transaction_failed",
			Network:     requirements.Network,
		}, nil
	}
	receipt, err := m.Signer.WaitForTransactionReceipt(ctx, txHash, requirements.Network, 120)
	if err != nil {
		return &x402.SettleResponse{
			Success:     false,
			Transaction: txHash,
			ErrorReason: "receipt_failed",
			Network:     requirements.Network,
		}, nil
	}
	status, _ := receipt["status"].(string)
	if status == "failed" {
		return &x402.SettleResponse{
			Success:     false,
			Transaction: txHash,
			ErrorReason: "transaction_failed_on_chain",
			Network:     requirements.Network,
		}, nil
	}
	return &x402.SettleResponse{
		Success:     true,
		Transaction: txHash,
		Network:     requirements.Network,
	}, nil
}

func (m *TransferAuthEvmFacilitatorMechanism) extractAuthorization(payload *x402.PaymentPayload) *TransferAuthorization {
	if payload.Extensions == nil {
		return nil
	}
	raw, ok := payload.Extensions["transferAuthorization"].(map[string]any)
	if !ok {
		return nil
	}
	from, _ := raw["from"].(string)
	to, _ := raw["to"].(string)
	value, _ := raw["value"].(string)
	validAfter, _ := raw["validAfter"].(string)
	validBefore, _ := raw["validBefore"].(string)
	nonce, _ := raw["nonce"].(string)
	if from == "" || to == "" || value == "" || nonce == "" {
		return nil
	}
	if validAfter == "" {
		validAfter = "0"
	}
	if validBefore == "" {
		validBefore = "0"
	}
	return &TransferAuthorization{
		From:        from,
		To:          to,
		Value:       value,
		ValidAfter:  validAfter,
		ValidBefore: validBefore,
		Nonce:       nonce,
	}
}

func (m *TransferAuthEvmFacilitatorMechanism) validateAuthorization(auth *TransferAuthorization, req x402.PaymentRequirements) string {
	adapter := m.Adapter
	if m.AllowedTokens != nil {
		if _, ok := m.AllowedTokens[adapter.NormalizeAddress(req.Asset)]; !ok {
			return "token_not_allowed"
		}
	}
	authVal, _ := strconv.ParseInt(auth.Value, 10, 64)
	reqVal, _ := strconv.ParseInt(req.Amount, 10, 64)
	if authVal < reqVal {
		return "amount_mismatch"
	}
	if adapter.NormalizeAddress(auth.To) != adapter.NormalizeAddress(req.PayTo) {
		return "payto_mismatch"
	}
	now := time.Now().Unix()
	validBefore, _ := strconv.ParseInt(auth.ValidBefore, 10, 64)
	if validBefore < now {
		return "expired"
	}
	validAfter, _ := strconv.ParseInt(auth.ValidAfter, 10, 64)
	if validAfter > now {
		return "not_yet_valid"
	}
	return ""
}

func (m *TransferAuthEvmFacilitatorMechanism) verifySignature(ctx context.Context, auth *TransferAuthorization, signature string, req x402.PaymentRequirements) (bool, error) {
	adapter := m.Adapter
	chainID, err := adapter.ParseChainID(req.Network)
	if err != nil {
		return false, err
	}
	tokenInfo := tokens.FindByAddress(req.Network, req.Asset)
	tokenName := "Unknown Token"
	tokenVersion := "1"
	if tokenInfo != nil {
		tokenName = tokenInfo.Name
		tokenVersion = tokenInfo.Version
		if tokenVersion == "" {
			tokenVersion = "1"
		}
	}
	domain := map[string]any{
		"name":              tokenName,
		"version":           tokenVersion,
		"chainId":           chainID,
		"verifyingContract": adapter.ToSigningAddress(req.Asset),
	}
	typesMap := typesToMapAny(TransferAuthEIP712Types())
	nonceBytes := HexToBytes(auth.Nonce)
	msgMap := map[string]any{
		"from":        auth.From,
		"to":          auth.To,
		"value":       auth.Value,
		"validAfter":  auth.ValidAfter,
		"validBefore": auth.ValidBefore,
		"nonce":       nonceBytes,
	}
	return m.Signer.VerifyTypedData(ctx, adapter.ToSigningAddress(auth.From), domain, typesMap, msgMap, signature, TransferAuthPrimaryType)
}

func intPtr64(i int64) *int64 {
	return &i
}
