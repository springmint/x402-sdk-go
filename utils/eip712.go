package utils

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	x402 "github.com/springmint/x402-sdk-go"
)

// PaymentIDToBytes converts payment ID from hex string to bytes16
func PaymentIDToBytes(paymentID string) ([]byte, error) {
	if !strings.HasPrefix(paymentID, "0x") {
		return nil, errors.New("invalid payment ID format: expected hex string with 0x prefix")
	}
	paymentIDHex := paymentID[2:]
	if len(paymentIDHex) != 32 {
		return nil, errors.New("invalid payment ID length: expected 32 hex characters")
	}
	return hex.DecodeString(paymentIDHex)
}

// EIP712Message represents the EIP-712 message structure for signing (matches Permit402Details)
type EIP712Message struct {
	Meta    EIP712PermitMeta
	Buyer   string
	Payment EIP712Payment
	Fee     EIP712Fee
}

type EIP712PermitMeta struct {
	Ptype      uint8
	PaymentID  []byte
	Nonce      *big.Int
	ValidAfter int64
	ValidBefore int64
}

type EIP712Payment struct {
	PayToken  string
	PayAmount *big.Int
	PayTo     string
}

type EIP712Fee struct {
	FeeTo     string
	FeeAmount *big.Int
}

// ConvertPermitToEIP712Message converts Permit402 to EIP-712 compatible struct
func ConvertPermitToEIP712Message(permit *x402.Permit402) (*EIP712Message, error) {
	paymentIDBytes, err := PaymentIDToBytes(permit.Meta.PaymentID)
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
	payAmount := new(big.Int)
	payAmount.SetString(permit.Payment.PayAmount, 10)
	feeAmount := new(big.Int)
	feeAmount.SetString(permit.Fee.FeeAmount, 10)
	return &EIP712Message{
		Meta: EIP712PermitMeta{
			Ptype:      ptype,
			PaymentID:  paymentIDBytes,
			Nonce:      nonce,
			ValidAfter: permit.Meta.ValidAfter,
			ValidBefore: permit.Meta.ValidBefore,
		},
		Buyer: permit.Buyer,
		Payment: EIP712Payment{
			PayToken:  permit.Payment.PayToken,
			PayAmount: payAmount,
			PayTo:     permit.Payment.PayTo,
		},
		Fee: EIP712Fee{
			FeeTo:     permit.Fee.FeeTo,
			FeeAmount: feeAmount,
		},
	}, nil
}
