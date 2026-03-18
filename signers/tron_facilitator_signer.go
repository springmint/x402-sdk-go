package signers

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/tron"
)

// TronFacilitatorSigner implements FacilitatorSigner for Tron
type TronFacilitatorSigner struct {
	privateKey *ecdsa.PrivateKey
	address    string
}

// NewTronFacilitatorSigner creates a TronFacilitatorSigner from private key hex
func NewTronFacilitatorSigner(privateKeyHex string) (*TronFacilitatorSigner, error) {
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
	return &TronFacilitatorSigner{
		privateKey: key,
		address:    addrBase58,
	}, nil
}

// GetAddress returns the signer's Tron address (base58)
func (s *TronFacilitatorSigner) GetAddress() string {
	return s.address
}

// VerifyTypedData verifies TIP-712 signature (same as EIP-712)
func (s *TronFacilitatorSigner) VerifyTypedData(ctx context.Context, address string, domain, types, message map[string]any, signature string, primaryType string) (bool, error) {
	log.Printf("[VerifyTypedData] domain: %v", domain)
	log.Printf("[VerifyTypedData] types: %v", types)
	log.Printf("[VerifyTypedData] message: %v", message)
	log.Printf("[VerifyTypedData] signature: %v", signature)
	log.Printf("[VerifyTypedData] primaryType: %v", primaryType)
	digest, err := EIP712Hash(domain, types, message, primaryType)
	if err != nil {
		log.Printf("[VerifyTypedData] EIP712Hash failed: %v", err)
		return false, err
	}
	log.Printf("[VerifyTypedData] digest: %s", hex.EncodeToString(digest))

	sig := signature
	if len(sig) >= 2 && sig[:2] == "0x" {
		sig = sig[2:]
	}
	sigBytes, err := hex.DecodeString(sig)
	if err != nil {
		log.Printf("[VerifyTypedData] hex.DecodeString(signature) failed: %v", err)
		return false, nil
	}
	if len(sigBytes) < 65 {
		log.Printf("[VerifyTypedData] signature too short: len=%d, need>=65", len(sigBytes))
		return false, nil
	}
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}
	pubKey, err := crypto.SigToPub(digest, sigBytes)
	if err != nil {
		log.Printf("[VerifyTypedData] SigToPub failed: %v", err)
		return false, nil
	}
	recovered := crypto.PubkeyToAddress(*pubKey)
	addrHex, err := tron.ToHex(address)
	if err != nil {
		log.Printf("[VerifyTypedData] address conversion failed (Tron base58->hex): %v", err)
		return false, err
	}
	expected := common.HexToAddress(addrHex)
	ok := strings.EqualFold(recovered.Hex(), expected.Hex())
	recoveredTron, _ := tron.HexToBase58(recovered.Hex())
	expectedTron, _ := tron.HexToBase58(expected.Hex())
	log.Printf("[VerifyTypedData] recovered=%s (Tron:%s) expected=%s (Tron:%s) match=%v", recovered.Hex(), recoveredTron, expected.Hex(), expectedTron, ok)
	return ok, nil
}

// WriteContract executes contract call via Tron TriggerSmartContract + sign + broadcast
func (s *TronFacilitatorSigner) WriteContract(ctx context.Context, contractAddress, abiJSON, method string, args []any, network string) (string, error) {
	apiURL := x402.GetRPCURL(network)
	if apiURL == "" {
		return "", x402.NewUnsupportedNetworkError("no RPC URL for Tron network: " + network)
	}
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", err
	}
	packed, err := parsed.Pack(method, args...)
	if err != nil {
		return "", err
	}
	if len(packed) < 4 {
		return "", x402.NewSettlementError("invalid packed data")
	}
	m, ok := parsed.Methods[method]
	if !ok {
		return "", x402.NewSettlementError("method not found: " + method)
	}
	parameterHex := hex.EncodeToString(packed[4:])
	tx, err := tron.TriggerContractCall(ctx, apiURL, contractAddress, s.address, m.Sig, parameterHex, 300_000_000)
	if err != nil {
		return "", err
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
		return "", err
	}
	if err := tron.BroadcastTransaction(ctx, apiURL, tx); err != nil {
		return "", err
	}
	return tx.TxID, nil
}

// WaitForTransactionReceipt polls Tron for transaction status
func (s *TronFacilitatorSigner) WaitForTransactionReceipt(ctx context.Context, txHash, network string, timeout int) (map[string]any, error) {
	apiURL := x402.GetRPCURL(network)
	if apiURL == "" {
		return nil, x402.NewUnsupportedNetworkError("no RPC URL for Tron network: " + network)
	}
	for i := 0; i < timeout; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		info, err := tron.GetTransactionInfo(ctx, apiURL, txHash)
		if err != nil {
			return nil, err
		}
		bn, _ := info["blockNumber"].(string)
		if bn != "" && bn != "0" {
			return info, nil
		}
		if i < timeout-1 {
			time.Sleep(time.Second)
		}
	}
	return nil, x402.NewSettlementError("transaction timeout")
}
