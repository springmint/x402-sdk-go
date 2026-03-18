package mechanisms

import (
	"context"
	"math/big"
	"strconv"
	"strings"
	"time"

	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/abi"
	"github.com/springmint/x402-sdk-go/signers"
	"github.com/springmint/x402-sdk-go/tokens"
	"github.com/springmint/x402-sdk-go/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/springmint/x402-sdk-go/tron"
)

func normalizeTronEvmAddress(addr string) (common.Address, error) {
	if strings.TrimSpace(addr) == "" {
		return common.Address{}, nil
	}

	// Accept either:
	// - Tron base58 (T...)
	// - EVM hex with 0x prefix
	// - EVM hex without 0x prefix
	a := strings.TrimSpace(addr)
	if strings.HasPrefix(a, "0x") {
		return common.HexToAddress(a), nil
	}
	if h, err := tron.ToHex(a); err == nil && h != "" {
		return common.HexToAddress(h), nil
	}
	// Fall back: treat as raw hex without prefix.
	return common.HexToAddress("0x" + a), nil
}

// Permit402TronFacilitatorMechanism implements permit402 facilitator for Tron
type Permit402TronFacilitatorMechanism struct {
	Signer  *signers.TronFacilitatorSigner
	FeeTo   string
	BaseFee map[string]int
}

// NewPermit402TronFacilitatorMechanism creates a Tron permit402 facilitator
func NewPermit402TronFacilitatorMechanism(signer *signers.TronFacilitatorSigner, feeTo string, baseFee map[string]int) *Permit402TronFacilitatorMechanism {
	if feeTo == "" {
		feeTo = signer.GetAddress()
	}
	return &Permit402TronFacilitatorMechanism{
		Signer:  signer,
		FeeTo:   feeTo,
		BaseFee: baseFee,
	}
}

// Scheme returns "permit402"
func (m *Permit402TronFacilitatorMechanism) Scheme() string {
	return SchemePermit402
}

// FeeQuote returns fee quote
func (m *Permit402TronFacilitatorMechanism) FeeQuote(ctx context.Context, accept x402.PaymentRequirements) (*x402.FeeQuoteResponse, error) {
	baseFee := m.getBaseFee(accept.Asset, accept.Network)
	if baseFee < 0 {
		return nil, nil
	}
	expires := time.Now().Unix() + FeeQuoteExpirySeconds
	feeToHex := m.FeeTo
	if h, err := tron.ToHex(m.FeeTo); err == nil {
		feeToHex = h
	}
	return &x402.FeeQuoteResponse{
		Fee: x402.FeeInfo{
			FeeTo:     feeToHex,
			FeeAmount: strconv.Itoa(baseFee),
		},
		Pricing:   "flat",
		Scheme:    accept.Scheme,
		Network:   accept.Network,
		Asset:     accept.Asset,
		ExpiresAt: &expires,
	}, nil
}

func (m *Permit402TronFacilitatorMechanism) getBaseFee(asset, network string) int {
	info := tokens.FindByAddress(network, asset)
	if info == nil {
		return -1
	}
	if m.BaseFee == nil {
		return 0
	}
	if fee, ok := m.BaseFee[strings.ToUpper(info.Symbol)]; ok {
		return fee
	}
	return 0
}

// Verify validates permit and TIP-712 signature
func (m *Permit402TronFacilitatorMechanism) Verify(ctx context.Context, payload *x402.PaymentPayload, requirements x402.PaymentRequirements) (*x402.VerifyResponse, error) {
	if payload.Payload.Permit402 == nil {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: "missing_permit402"}, nil
	}
	permit := payload.Payload.Permit402
	if err := m.validatePermit(permit, requirements); err != "" {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: err}, nil
	}
	chainID, err := x402.GetChainID(requirements.Network)
	if err != nil {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: "unsupported_network"}, nil
	}
	permitAddr := x402.GetPermit402Address(requirements.Network)
	permitAddrHex, err := tron.ToHex(permitAddr)
	if err != nil {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: "invalid_permit_addr"}, nil
	}
	domain := map[string]any{
		"name":              "Permit402",
		"chainId":           chainID,
		"verifyingContract": permitAddrHex,
	}
	typesMap := abiTypesToMap(abi.GetPermit402EIP712Types())
	eip712Msg, err := utils.ConvertPermitToEIP712Message(permit)
	if err != nil {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: "invalid_permit"}, nil
	}
	msgMap := eip712MessageToMap(eip712Msg)
	ok, err := m.Signer.VerifyTypedData(ctx, permit.Buyer, domain, typesMap, msgMap, payload.Payload.Signature, abi.Permit402PrimaryType)
	if err != nil || !ok {
		return &x402.VerifyResponse{IsValid: false, InvalidReason: "invalid_signature"}, nil
	}
	return &x402.VerifyResponse{IsValid: true}, nil
}

func (m *Permit402TronFacilitatorMechanism) validatePermit(permit *x402.Permit402, req x402.PaymentRequirements) string {
	norm := func(addr string) string {
		if addr == "" {
			return ""
		}
		h, err := tron.ToHex(addr)
		if err != nil {
			return strings.ToLower(addr)
		}
		return strings.ToLower(h)
	}
	payAmt := new(big.Int)
	payAmt.SetString(permit.Payment.PayAmount, 10)
	reqAmt := new(big.Int)
	reqAmt.SetString(req.Amount, 10)
	if payAmt.Cmp(reqAmt) < 0 {
		return "amount_mismatch"
	}
	if norm(permit.Payment.PayTo) != norm(req.PayTo) {
		return "payto_mismatch"
	}
	if norm(permit.Payment.PayToken) != norm(req.Asset) {
		return "token_mismatch"
	}
	zeroAddr := strings.ToLower(x402.ZeroAddress)
	feeToNorm := norm(permit.Fee.FeeTo)
	if feeToNorm != zeroAddr {
		if feeToNorm != norm(m.FeeTo) {
			return "fee_to_mismatch"
		}
		baseFee := m.getBaseFee(permit.Payment.PayToken, req.Network)
		if baseFee < 0 {
			return "unsupported_token"
		}
		feeAmt, _ := strconv.ParseInt(permit.Fee.FeeAmount, 10, 64)
		if feeAmt < int64(baseFee) {
			return "fee_amount_mismatch"
		}
	}
	now := time.Now().Unix()
	if permit.Meta.ValidBefore < now {
		return "expired"
	}
	if permit.Meta.ValidAfter > now {
		return "not_yet_valid"
	}
	return ""
}

// Settle calls permitTransferFrom on Tron Permit402 contract
func (m *Permit402TronFacilitatorMechanism) Settle(ctx context.Context, payload *x402.PaymentPayload, requirements x402.PaymentRequirements) (*x402.SettleResponse, error) {
	resp, err := m.Verify(ctx, payload, requirements)
	if err != nil || !resp.IsValid {
		return &x402.SettleResponse{
			Success:     false,
			ErrorReason: resp.InvalidReason,
			Network:     requirements.Network,
		}, nil
	}
	permit := payload.Payload.Permit402
	permitArg, err := m.buildPermitArg(permit)
	if err != nil {
		return &x402.SettleResponse{
			Success:     false,
			ErrorReason: "invalid_permit",
			Network:     requirements.Network,
		}, nil
	}
	sig := payload.Payload.Signature
	if len(sig) >= 2 && sig[:2] == "0x" {
		sig = sig[2:]
	}
	sigBytes, _ := hexDecode(sig)
	buyerAddr, err := normalizeTronEvmAddress(permit.Buyer)
	if err != nil {
		return &x402.SettleResponse{
			Success:     false,
			ErrorReason: "invalid_buyer",
			Network:     requirements.Network,
		}, nil
	}
	contractAddr := x402.GetPermit402Address(requirements.Network)
	args := []any{*permitArg, buyerAddr, sigBytes}
	txHash, err := m.Signer.WriteContract(ctx, contractAddr, abi.Permit402ABI, "permitTransferFrom", args, requirements.Network)
	if err != nil || txHash == "" {
		errMsg := "transaction_failed"
		if err != nil {
			errMsg = errMsg + ": " + err.Error()
		}
		return &x402.SettleResponse{
			Success:     false,
			ErrorReason: errMsg,
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

func (m *Permit402TronFacilitatorMechanism) buildPermitArg(permit *x402.Permit402) (*permitArg, error) {
	paymentIDBytes, err := utils.PaymentIDToBytes(permit.Meta.PaymentID)
	if err != nil {
		return nil, err
	}
	ptype := uint8(0)
	if k, ok := x402.PtypeMap[permit.Meta.Ptype]; ok {
		ptype = k
	}
	nonce := new(big.Int)
	if permit.Meta.Nonce != "" {
		nonce.SetString(permit.Meta.Nonce, 10)
	}
	var pid [16]byte
	copy(pid[:], paymentIDBytes)
	meta := permitMetaArg{
		Ptype:       ptype,
		PaymentID:   pid,
		Nonce:       nonce,
		ValidAfter:  big.NewInt(permit.Meta.ValidAfter),
		ValidBefore: big.NewInt(permit.Meta.ValidBefore),
	}
	payAmount := new(big.Int)
	payAmount.SetString(permit.Payment.PayAmount, 10)
	feeAmount := new(big.Int)
	feeAmount.SetString(permit.Fee.FeeAmount, 10)

	payTokenAddr, err := normalizeTronEvmAddress(permit.Payment.PayToken)
	if err != nil {
		return nil, err
	}
	payToAddr, err := normalizeTronEvmAddress(permit.Payment.PayTo)
	if err != nil {
		return nil, err
	}
	feeToAddr, err := normalizeTronEvmAddress(permit.Fee.FeeTo)
	if err != nil {
		return nil, err
	}
	buyerAddr, err := normalizeTronEvmAddress(permit.Buyer)
	if err != nil {
		return nil, err
	}

	payment := paymentArg{
		PayToken:  payTokenAddr,
		PayAmount: payAmount,
		PayTo:     payToAddr,
	}
	fee := feeArg{
		FeeTo:     feeToAddr,
		FeeAmount: feeAmount,
	}
	return &permitArg{
		Meta:    meta,
		Buyer:   buyerAddr,
		Payment: payment,
		Fee:     fee,
	}, nil
}
