package mechanisms

import (
	"context"
	"math/big"

	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/abi"
	"github.com/springmint/x402-sdk-go/signers"
	"github.com/springmint/x402-sdk-go/utils"
)

// Permit402EvmClientMechanism implements permit402 client for EVM
type Permit402EvmClientMechanism struct {
	Signer signers.ClientSigner
}

// Scheme returns "permit402"
func (m *Permit402EvmClientMechanism) Scheme() string {
	return "permit402"
}

// CreatePaymentPayload creates signed PaymentPayload
func (m *Permit402EvmClientMechanism) CreatePaymentPayload(ctx context.Context, requirements x402.PaymentRequirements, resource string, extensions map[string]any) (*x402.PaymentPayload, error) {
	ctxMeta, ok := getContextMeta(extensions)
	if !ok {
		return nil, x402.NewPermitValidationError("missing_context", "permit402Context is required")
	}

	feeTo := x402.ZeroAddress
	feeAmount := "0"

	if requirements.Extra != nil && requirements.Extra.Fee != nil {
		if requirements.Extra.Fee.FeeTo != "" {
			feeTo = requirements.Extra.Fee.FeeTo
		}
		if requirements.Extra.Fee.FeeAmount != "" {
			feeAmount = requirements.Extra.Fee.FeeAmount
		}
	}

	permit := &x402.Permit402{
		Meta: x402.PermitMeta{
			Ptype:       getStr(ctxMeta, "ptype"),
			PaymentID:   getStr(ctxMeta, "paymentId"),
			Nonce:       getStr(ctxMeta, "nonce"),
			ValidAfter:  getInt64(ctxMeta, "validAfter"),
			ValidBefore: getInt64(ctxMeta, "validBefore"),
		},
		Buyer: m.Signer.GetAddress(),
		Payment: x402.Payment{
			PayToken:  requirements.Asset,
			PayAmount: requirements.Amount,
			PayTo:     requirements.PayTo,
		},
		Fee: x402.Fee{FeeTo: feeTo, FeeAmount: feeAmount},
	}

	payAmt := new(big.Int)
	payAmt.SetString(requirements.Amount, 10)
	feeAmt := new(big.Int)
	feeAmt.SetString(feeAmount, 10)
	totalAmount := new(big.Int).Add(payAmt, feeAmt)
	_, err := m.Signer.EnsureAllowance(ctx, requirements.Asset, totalAmount, requirements.Network, "auto")
	if err != nil {
		return nil, err
	}

	eip712Msg, err := utils.ConvertPermitToEIP712Message(permit)
	if err != nil {
		return nil, err
	}

	chainID, err := x402.GetChainID(requirements.Network)
	if err != nil {
		return nil, err
	}
	permitAddr := x402.GetPermit402Address(requirements.Network)
	domain := map[string]any{
		"name":              "Permit402",
		"chainId":           chainID,
		"verifyingContract": permitAddr,
	}
	typesMap := abiTypesToMap(abi.GetPermit402EIP712Types())
	msgMap := eip712MessageToMap(eip712Msg)

	signature, err := m.Signer.SignTypedData(ctx, domain, typesMap, msgMap, abi.Permit402PrimaryType)
	if err != nil {
		return nil, err
	}

	return &x402.PaymentPayload{
		X402Version: 2,
		Resource:    &x402.ResourceInfo{URL: resource},
		Accepted:    requirements,
		Payload: x402.PaymentPayloadData{
			Signature:     signature,
			Permit402: permit,
		},
		Extensions: map[string]any{},
	}, nil
}

func getContextMeta(ext map[string]any) (map[string]any, bool) {
	if ext == nil {
		return nil, false
	}
	ctx, ok := ext["permit402Context"].(map[string]any)
	if !ok {
		return nil, false
	}
	meta, ok := ctx["meta"].(map[string]any)
	if !ok {
		return nil, false
	}
	return meta, true
}

func getStr(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func getInt64(m map[string]any, k string) int64 {
	switch v := m[k].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

func abiTypesToMap(t map[string][]map[string]string) map[string]any {
	out := make(map[string]any)
	for k, v := range t {
		out[k] = v
	}
	return out
}

func eip712MessageToMap(msg *utils.EIP712Message) map[string]any {
	payAmount := "0"
	if msg.Payment.PayAmount != nil {
		payAmount = msg.Payment.PayAmount.String()
	}
	feeAmount := "0"
	if msg.Fee.FeeAmount != nil {
		feeAmount = msg.Fee.FeeAmount.String()
	}
	nonce := "0"
	if msg.Meta.Nonce != nil {
		nonce = msg.Meta.Nonce.String()
	}
	return map[string]any{
		"meta": map[string]any{
			"ptype":      msg.Meta.Ptype,
			"paymentId":  msg.Meta.PaymentID,
			"nonce":      nonce,
			"validAfter": msg.Meta.ValidAfter,
			"validBefore": msg.Meta.ValidBefore,
		},
		"buyer": msg.Buyer,
		"payment": map[string]any{
			"payToken":  msg.Payment.PayToken,
			"payAmount": payAmount,
			"payTo":     msg.Payment.PayTo,
		},
		"fee": map[string]any{
			"feeTo":     msg.Fee.FeeTo,
			"feeAmount": feeAmount,
		},
	}
}
