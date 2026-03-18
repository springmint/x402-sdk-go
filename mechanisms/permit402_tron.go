package mechanisms

import (
	"context"
	"math/big"

	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/abi"
	"github.com/springmint/x402-sdk-go/signers"
	"github.com/springmint/x402-sdk-go/tron"
	"github.com/springmint/x402-sdk-go/utils"
)

// Permit402TronClientMechanism implements permit402 client for Tron (TIP-712)
type Permit402TronClientMechanism struct {
	Signer *signers.TronClientSigner
}

// Scheme returns "permit402"
func (m *Permit402TronClientMechanism) Scheme() string {
	return "permit402"
}

// CreatePaymentPayload creates signed PaymentPayload for Tron
func (m *Permit402TronClientMechanism) CreatePaymentPayload(ctx context.Context, requirements x402.PaymentRequirements, resource string, extensions map[string]any) (*x402.PaymentPayload, error) {
	ctxMeta, ok := getContextMeta(extensions)
	if !ok {
		return nil, x402.NewPermitValidationError("missing_context", "permit402Context is required")
	}

	feeTo := x402.ZeroAddress
	feeAmount := "0"

	if requirements.Extra != nil && requirements.Extra.Fee != nil {
		if requirements.Extra.Fee.FeeTo != "" {
			h, err := tron.ToHex(requirements.Extra.Fee.FeeTo)
			if err != nil {
				return nil, x402.NewPermitValidationError("invalid_fee_to", err.Error())
			}
			feeTo = h
		}
		if requirements.Extra.Fee.FeeAmount != "" {
			feeAmount = requirements.Extra.Fee.FeeAmount
		}
	}

	// Convert addresses to hex for permit (TIP-712). Accept both base58 (T...) and hex (0x...)
	buyerHex := m.Signer.GetAddressHex()
	payTokenHex, err := tron.ToHex(requirements.Asset)
	if err != nil {
		return nil, x402.NewPermitValidationError("invalid_asset", err.Error())
	}
	payToHex, err := tron.ToHex(requirements.PayTo)
	if err != nil {
		return nil, x402.NewPermitValidationError("invalid_pay_to", err.Error())
	}

	permit := &x402.Permit402{
		Meta: x402.PermitMeta{
			Ptype:       getStr(ctxMeta, "ptype"),
			PaymentID:   getStr(ctxMeta, "paymentId"),
			Nonce:       getStr(ctxMeta, "nonce"),
			ValidAfter:  getInt64(ctxMeta, "validAfter"),
			ValidBefore: getInt64(ctxMeta, "validBefore"),
		},
		Buyer: buyerHex,
		Payment: x402.Payment{
			PayToken:  payTokenHex,
			PayAmount: requirements.Amount,
			PayTo:     payToHex,
		},
		Fee: x402.Fee{FeeTo: feeTo, FeeAmount: feeAmount},
	}

	payAmt := new(big.Int)
	payAmt.SetString(requirements.Amount, 10)
	feeAmt := new(big.Int)
	feeAmt.SetString(feeAmount, 10)
	totalAmount := new(big.Int).Add(payAmt, feeAmt)
	_, err = m.Signer.EnsureAllowance(ctx, requirements.Asset, totalAmount, requirements.Network, "auto")
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
	permitAddrHex, err := tron.ToHex(permitAddr)
	if err != nil {
		return nil, x402.NewPermitValidationError("invalid_permit_addr", err.Error())
	}

	domain := map[string]any{
		"name":              "Permit402",
		"chainId":           chainID,
		"verifyingContract": permitAddrHex,
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
			Signature: signature,
			Permit402: permit,
		},
		Extensions: map[string]any{},
	}, nil
}
