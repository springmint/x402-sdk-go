package signers

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	x402 "github.com/springmint/x402-sdk-go"
	"github.com/springmint/x402-sdk-go/abi"
)

// EvmClientSigner implements ClientSigner for EVM
type EvmClientSigner struct {
	privateKey *ecdsa.PrivateKey
	address    string
	clients    map[string]*ethclient.Client
}

// NewEvmClientSigner creates an EvmClientSigner from private key hex
func NewEvmClientSigner(privateKeyHex string) (*EvmClientSigner, error) {
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return &EvmClientSigner{
		privateKey: key,
		address:    addr.Hex(),
		clients:    make(map[string]*ethclient.Client),
	}, nil
}

// GetAddress returns the signer's address
func (s *EvmClientSigner) GetAddress() string {
	return s.address
}

// getClient returns or creates ethclient for network
func (s *EvmClientSigner) getClient(ctx context.Context, network string) (*ethclient.Client, error) {
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

// SignMessage signs raw message (EIP-191)
func (s *EvmClientSigner) SignMessage(ctx context.Context, message []byte) (string, error) {
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

// SignTypedData signs EIP-712 typed data
func (s *EvmClientSigner) SignTypedData(ctx context.Context, domain, types, message map[string]any, primaryType string) (string, error) {
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

// CheckBalance returns token balance
func (s *EvmClientSigner) CheckBalance(ctx context.Context, token, network string) (*big.Int, error) {
	client, err := s.getClient(ctx, network)
	if err != nil {
		return nil, err
	}
	return callBalanceOf(ctx, client, common.HexToAddress(token), common.HexToAddress(s.address))
}

// CheckAllowance returns allowance for spender
func (s *EvmClientSigner) CheckAllowance(ctx context.Context, token string, amount *big.Int, network string) (*big.Int, error) {
	_ = amount
	client, err := s.getClient(ctx, network)
	if err != nil {
		return nil, err
	}
	spender := x402.GetPermit402Address(network)
	return callAllowance(ctx, client, common.HexToAddress(token), common.HexToAddress(s.address), common.HexToAddress(spender))
}

// EnsureAllowance ensures sufficient allowance for spender
func (s *EvmClientSigner) EnsureAllowance(ctx context.Context, token string, amount *big.Int, network string, mode string) (bool, error) {
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
	client, err := s.getClient(ctx, network)
	if err != nil {
		return false, err
	}
	chainID, err := x402.GetChainID(network)
	if err != nil {
		return false, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(s.privateKey, big.NewInt(chainID))
	if err != nil {
		return false, x402.NewInsufficientAllowanceError(err.Error())
	}
	spender := x402.GetPermit402Address(network)
	maxApproval := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	maxApproval.Sub(maxApproval, big.NewInt(1))
	_, err = transactApprove(ctx, client, common.HexToAddress(token), common.HexToAddress(spender), maxApproval, auth)
	if err != nil {
		return false, err
	}
	return true, nil
}

// transactApprove sends ERC20 approve transaction
func transactApprove(ctx context.Context, client *ethclient.Client, tokenAddr, spender common.Address, amount *big.Int, auth *bind.TransactOpts) (string, error) {
	contract := bind.NewBoundContract(tokenAddr, abi.ERC20ABI, client, client, client)
	tx, err := contract.Transact(auth, "approve", spender, amount)
	if err != nil {
		return "", x402.NewInsufficientAllowanceError(err.Error())
	}
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return "", x402.NewInsufficientAllowanceError(err.Error())
	}
	if receipt.Status != 1 {
		return "", x402.NewInsufficientAllowanceError("approve transaction failed")
	}
	return tx.Hash().Hex(), nil
}

// callBalanceOf invokes ERC20 balanceOf
func callBalanceOf(ctx context.Context, client *ethclient.Client, token, account common.Address) (*big.Int, error) {
	data := common.FromHex("70a08231")
	data = append(data, common.LeftPadBytes(account.Bytes(), 32)...)
	to := token
	msg := ethereum.CallMsg{To: &to, Data: data}
	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(result), nil
}

// callAllowance invokes ERC20 allowance
func callAllowance(ctx context.Context, client *ethclient.Client, token, owner, spender common.Address) (*big.Int, error) {
	data := common.FromHex("dd62ed3e")
	data = append(data, common.LeftPadBytes(owner.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(spender.Bytes(), 32)...)
	to := token
	msg := ethereum.CallMsg{To: &to, Data: data}
	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(result), nil
}
