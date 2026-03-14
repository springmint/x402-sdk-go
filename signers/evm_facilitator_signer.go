package signers

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"time"

	x402 "github.com/springmint/x402-sdk-go"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EvmFacilitatorSigner implements FacilitatorSigner for EVM
type EvmFacilitatorSigner struct {
	privateKey *ecdsa.PrivateKey
	address    string
	clients    map[string]*ethclient.Client
}

// NewEvmFacilitatorSigner creates an EvmFacilitatorSigner from private key hex
func NewEvmFacilitatorSigner(privateKeyHex string) (*EvmFacilitatorSigner, error) {
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return &EvmFacilitatorSigner{
		privateKey: key,
		address:    addr.Hex(),
		clients:    make(map[string]*ethclient.Client),
	}, nil
}

// GetAddress returns the signer's address
func (s *EvmFacilitatorSigner) GetAddress() string {
	return s.address
}

func (s *EvmFacilitatorSigner) getClient(ctx context.Context, network string) (*ethclient.Client, error) {
	if c, ok := s.clients[network]; ok && c != nil {
		return c, nil
	}
	rpcURL := x402.GetRPCURL(network)
	if rpcURL == "" {
		return nil, x402.NewUnsupportedNetworkError("no RPC URL for network: " + network)
	}
	c, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	s.clients[network] = c
	return c, nil
}

// VerifyTypedData verifies EIP-712 signature
func (s *EvmFacilitatorSigner) VerifyTypedData(ctx context.Context, address string, domain, types, message map[string]any, signature string, primaryType string) (bool, error) {
	digest, err := EIP712Hash(domain, types, message, primaryType)
	if err != nil {
		return false, err
	}
	sig := signature
	if len(sig) >= 2 && sig[:2] == "0x" {
		sig = sig[2:]
	}
	sigBytes := common.FromHex(sig)
	if len(sigBytes) < 65 {
		return false, nil
	}
	// Adjust V if needed
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}
	pubKey, err := crypto.SigToPub(digest, sigBytes)
	if err != nil {
		return false, nil
	}
	recovered := crypto.PubkeyToAddress(*pubKey)
	expected := common.HexToAddress(address)
	return recovered.Hex() == expected.Hex(), nil
}

// WriteContract executes contract transaction (e.g. transferWithAuthorization on token, or permitTransferFrom on Permit402).
func (s *EvmFacilitatorSigner) WriteContract(ctx context.Context, contractAddress, abiJSON, method string, args []any, network string) (string, error) {
	client, err := s.getClient(ctx, network)
	if err != nil {
		return "", err
	}
	contractABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", err
	}
	to := common.HexToAddress(contractAddress)
	chainID := big.NewInt(0)
	if id, err := x402.GetChainID(network); err == nil {
		chainID.SetInt64(id)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(s.privateKey, chainID)
	if err != nil {
		return "", err
	}
	auth.Context = ctx
	bound := bind.NewBoundContract(to, contractABI, client, client, nil)
	tx, err := bound.Transact(auth, method, args...)
	if err != nil {
		return "", err
	}
	if tx == nil {
		return "", x402.NewSettlementError("no tx returned")
	}
	return tx.Hash().Hex(), nil
}

// WaitForTransactionReceipt waits for tx confirmation
func (s *EvmFacilitatorSigner) WaitForTransactionReceipt(ctx context.Context, txHash, network string, timeout int) (map[string]any, error) {
	client, err := s.getClient(ctx, network)
	if err != nil {
		return nil, err
	}
	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(txHash))
	for err != nil && timeout > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			time.Sleep(time.Second)
			timeout--
		}
		receipt, err = client.TransactionReceipt(ctx, common.HexToHash(txHash))
	}
	if err != nil {
		return nil, err
	}
	status := "confirmed"
	if receipt.Status == 0 {
		status = "failed"
	}
	return map[string]any{
		"hash":        txHash,
		"blockNumber": receipt.BlockNumber.String(),
		"status":      status,
	}, nil
}
