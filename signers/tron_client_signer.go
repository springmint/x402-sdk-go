package signers

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/tron"
)

// TronClientSigner implements ClientSigner for Tron (TIP-712)
type TronClientSigner struct {
	privateKey *ecdsa.PrivateKey
	address    string // base58 Tron address
}

// NewTronClientSigner creates a TronClientSigner from private key hex.
// Uses same key format as Ethereum (secp256k1).
func NewTronClientSigner(privateKeyHex string) (*TronClientSigner, error) {
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}
	addrHex := crypto.PubkeyToAddress(key.PublicKey).Hex()
	addrBase58, err := tron.HexToBase58(addrHex)
	if err != nil {
		return nil, err
	}
	return &TronClientSigner{
		privateKey: key,
		address:    addrBase58,
	}, nil
}

// GetAddress returns the signer's Tron address (base58)
func (s *TronClientSigner) GetAddress() string {
	return s.address
}

// GetAddressHex returns the 20-byte address as 0x-prefixed hex (for EIP-712/TIP-712)
func (s *TronClientSigner) GetAddressHex() string {
	h, _ := tron.Base58ToHex(s.address)
	return h
}

// SignMessage signs raw message (EIP-191 style)
func (s *TronClientSigner) SignMessage(ctx context.Context, message []byte) (string, error) {
	prefixed := crypto.Keccak256Hash(
		append([]byte("\x19Ethereum Signed Message:\n32"), crypto.Keccak256Hash(message).Bytes()...),
	)
	sig, err := crypto.Sign(prefixed.Bytes(), s.privateKey)
	if err != nil {
		return "", x402.NewSignatureCreationError(err.Error())
	}
	sig[64] += 27
	return "0x" + common.Bytes2Hex(sig), nil
}

// SignTypedData signs TIP-712 typed data (same as EIP-712)
// Domain and message must use hex addresses (0x + 40 chars) for address fields
func (s *TronClientSigner) SignTypedData(ctx context.Context, domain, types, message map[string]any, primaryType string) (string, error) {
	digest, err := EIP712Hash(domain, types, message, primaryType)
	if err != nil {
		return "", x402.NewSignatureCreationError(err.Error())
	}
	sig, err := crypto.Sign(digest, s.privateKey)
	if err != nil {
		return "", x402.NewSignatureCreationError(err.Error())
	}
	sig[64] += 27
	return "0x" + common.Bytes2Hex(sig), nil
}

// CheckBalance returns TRC20 token balance
func (s *TronClientSigner) CheckBalance(ctx context.Context, token, network string) (*big.Int, error) {
	apiURL := x402.GetRPCURL(network)
	if apiURL == "" {
		return nil, x402.NewUnsupportedNetworkError("no RPC URL for Tron network: " + network)
	}
	return tron.CallBalanceOf(ctx, apiURL, token, s.address)
}

// CheckAllowance returns TRC20 allowance for spender
func (s *TronClientSigner) CheckAllowance(ctx context.Context, token string, amount *big.Int, network string) (*big.Int, error) {
	_ = amount
	apiURL := x402.GetRPCURL(network)
	if apiURL == "" {
		return nil, x402.NewUnsupportedNetworkError("no RPC URL for Tron network: " + network)
	}
	spender := x402.GetPermit402Address(network)
	return tron.CallAllowance(ctx, apiURL, token, s.address, spender)
}

// EnsureAllowance ensures sufficient TRC20 allowance; performs approve tx if needed
func (s *TronClientSigner) EnsureAllowance(ctx context.Context, token string, amount *big.Int, network string, mode string) (bool, error) {
	if mode == "skip" {
		return true, nil
	}
	current, err := s.CheckAllowance(ctx, token, amount, network)
	if err != nil {
		return false, err
	}
	if current.Cmp(amount) >= 0 {
		return true, nil
	}
	if mode == "interactive" {
		return false, x402.NewInsufficientAllowanceError("interactive approval required")
	}

	apiURL := x402.GetRPCURL(network)
	if apiURL == "" {
		return false, x402.NewUnsupportedNetworkError("no RPC URL for Tron network: " + network)
	}
	spender := x402.GetPermit402Address(network)
	maxApproval := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	maxApproval.Sub(maxApproval, big.NewInt(1))

	tx, err := tron.TriggerApprove(ctx, apiURL, token, s.address, spender, maxApproval)
	if err != nil {
		return false, x402.NewInsufficientAllowanceError("tron approve: " + err.Error())
	}
	signHash := func(hash []byte) ([]byte, error) {
		sig, err := crypto.Sign(hash, s.privateKey)
		if err != nil {
			return nil, err
		}
		sig[64] += 27
		return sig, nil
	}
	if err := tron.SignTransaction(tx, signHash); err != nil {
		return false, x402.NewInsufficientAllowanceError("tron sign: " + err.Error())
	}
	if err := tron.BroadcastTransaction(ctx, apiURL, tx); err != nil {
		return false, x402.NewInsufficientAllowanceError("tron broadcast: " + err.Error())
	}
	return true, nil
}
