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
)

const (
	SchemePermit402   = "permit402"
	FeeQuoteExpirySeconds = 300
)

// Permit402 contract error selectors for debugging
var Permit402Errors = map[string]string{
	"0xb7d09497": "InvalidTimestamp",
	"0x756688fe": "InvalidNonce",
	"0x8baa579f": "InvalidSignature",
	"0x1fb09b80": "NonceAlreadyUsed",
	"0x2b79ed30": "InvalidPtype",
	"0xadf31a79": "BuyerMismatch",
}

func DecodePermit402Error(data string) string {
	if len(data) >= 10 {
		selector := data[:10]
		if errName, ok := Permit402Errors[selector]; ok {
			return errName
		}
	}
	return "Unknown error: " + data
}

// Permit402EvmFacilitatorMechanism implements permit402 facilitator for EVM
type Permit402EvmFacilitatorMechanism struct {
	Signer  signers.FacilitatorSigner
	FeeTo   string
	BaseFee map[string]int // symbol -> amount in smallest units
}

// NewPermit402EvmFacilitatorMechanism creates a Permit402 EVM facilitator
func NewPermit402EvmFacilitatorMechanism(signer signers.FacilitatorSigner, feeTo string, baseFee map[string]int) *Permit402EvmFacilitatorMechanism {
	if feeTo == "" {
		feeTo = signer.GetAddress()
	}
	return &Permit402EvmFacilitatorMechanism{
		Signer:  signer,
		FeeTo:   feeTo,
		BaseFee: baseFee,
	}
}

// Scheme returns "permit402"
func (m *Permit402EvmFacilitatorMechanism) Scheme() string {
	return SchemePermit402
}

// FeeQuote returns fee quote for the given requirements
func (m *Permit402EvmFacilitatorMechanism) FeeQuote(ctx context.Context, accept x402.PaymentRequirements) (*x402.FeeQuoteResponse, error) {
	baseFee := m.getBaseFee(accept.Asset, accept.Network)
	if baseFee < 0 {
		return nil, nil
	}
	expires := time.Now().Unix() + FeeQuoteExpirySeconds
	return &x402.FeeQuoteResponse{
		Fee: x402.FeeInfo{
			FeeTo:     m.FeeTo,
			FeeAmount: strconv.Itoa(baseFee),
		},
		Pricing:   "flat",
		Scheme:    accept.Scheme,
		Network:   accept.Network,
		Asset:     accept.Asset,
		ExpiresAt: &expires,
	}, nil
}

func (m *Permit402EvmFacilitatorMechanism) getBaseFee(asset, network string) int {
	info := tokens.FindByAddress(network, asset)
	if info == nil {
		return -1
	}
	symbol := strings.ToUpper(info.Symbol)
	if m.BaseFee == nil {
		return 0
	}
	if fee, ok := m.BaseFee[symbol]; ok {
		return fee
	}
	return 0
}

// Verify validates permit and EIP-712 signature
func (m *Permit402EvmFacilitatorMechanism) Verify(ctx context.Context, payload *x402.PaymentPayload, requirements x402.PaymentRequirements) (*x402.VerifyResponse, error) {
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
	domain := map[string]any{
		"name":              "Permit402",
		"chainId":           chainID,
		"verifyingContract": permitAddr,
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

func (m *Permit402EvmFacilitatorMechanism) validatePermit(permit *x402.Permit402, req x402.PaymentRequirements) string {
	norm := strings.ToLower
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
	zeroAddr := x402.ZeroAddress
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

// Settle calls permitTransferFrom on Permit402 contract
func (m *Permit402EvmFacilitatorMechanism) Settle(ctx context.Context, payload *x402.PaymentPayload, requirements x402.PaymentRequirements) (*x402.SettleResponse, error) {
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
	buyer := normAddr(permit.Buyer)
	contractAddr := x402.GetPermit402Address(requirements.Network)
	args := []any{*permitArg, common.HexToAddress(buyer), sigBytes}
	txHash, err := m.Signer.WriteContract(ctx, contractAddr, abi.Permit402ABI, "permitTransferFrom", args, requirements.Network)
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

// Types matching Permit402 tuple ABI (IPermit402.Permit402Details)
type permitMetaArg struct {
	Ptype       uint8    `abi:"ptype"`
	PaymentID   [16]byte `abi:"paymentId"`
	Nonce       *big.Int `abi:"nonce"`
	ValidAfter  *big.Int `abi:"validAfter"`
	ValidBefore *big.Int `abi:"validBefore"`
}

type paymentArg struct {
	PayToken  common.Address `abi:"payToken"`
	PayAmount *big.Int       `abi:"payAmount"`
	PayTo     common.Address `abi:"payTo"`
}

type feeArg struct {
	FeeTo     common.Address `abi:"feeTo"`
	FeeAmount *big.Int       `abi:"feeAmount"`
}

type permitArg struct {
	Meta    permitMetaArg  `abi:"meta"`
	Buyer   common.Address `abi:"buyer"`
	Payment paymentArg     `abi:"payment"`
	Fee     feeArg         `abi:"fee"`
}

// buildPermitArg converts Permit402 into the struct matching Permit402ABI tuple.
func (m *Permit402EvmFacilitatorMechanism) buildPermitArg(permit *x402.Permit402) (*permitArg, error) {
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
	payment := paymentArg{
		PayToken:  common.HexToAddress(permit.Payment.PayToken),
		PayAmount: payAmount,
		PayTo:     common.HexToAddress(permit.Payment.PayTo),
	}
	fee := feeArg{
		FeeTo:     common.HexToAddress(permit.Fee.FeeTo),
		FeeAmount: feeAmount,
	}
	out := &permitArg{
		Meta:    meta,
		Buyer:   common.HexToAddress(permit.Buyer),
		Payment: payment,
		Fee:     fee,
	}
	return out, nil
}

func commonHex(addr string) string {
	if len(addr) >= 2 && addr[:2] == "0x" {
		return addr
	}
	return "0x" + addr
}

func normAddr(addr string) string {
	return strings.ToLower(commonHex(addr))
}

func hexDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var n byte
		for _, c := range s[i : i+2] {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n |= byte(c - '0')
			case c >= 'a' && c <= 'f':
				n |= byte(c - 'a' + 10)
			case c >= 'A' && c <= 'F':
				n |= byte(c - 'A' + 10)
			}
		}
		b[i/2] = n
	}
	return b, nil
}

