package tron

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
)


// TriggerConstantContractReq is the request body for Tron TriggerConstantContract API
type TriggerConstantContractReq struct {
	OwnerAddress     string `json:"owner_address"`
	ContractAddress  string `json:"contract_address"`
	FunctionSelector string `json:"function_selector"`
	Parameter        string `json:"parameter"`
	Visible          bool   `json:"visible,omitempty"`
}

// TriggerSmartContractReq is the request for TriggerSmartContract (state-changing)
type TriggerSmartContractReq struct {
	OwnerAddress     string `json:"owner_address"`
	ContractAddress  string `json:"contract_address"`
	FunctionSelector string `json:"function_selector"`
	Parameter        string `json:"parameter"`
	FeeLimit        int64  `json:"fee_limit"`
	Visible         bool   `json:"visible,omitempty"`
}

// TronTransaction is the transaction from TriggerSmartContract response
type TronTransaction struct {
	RawData    json.RawMessage `json:"raw_data"`
	RawDataHex string          `json:"raw_data_hex"`
	TxID       string          `json:"txID"`
	Signature  []string        `json:"signature,omitempty"`
}

// TriggerSmartContractResp is the response
type TriggerSmartContractResp struct {
	Result       *TronTransaction `json:"result"`
	Transaction  *TronTransaction `json:"transaction"`
	Transaction2 *TronTransaction `json:"Transaction"`
	Code         string          `json:"code,omitempty"`
	Message      string          `json:"message,omitempty"`
}

// BroadcastResp is the response from broadcasttransaction
type BroadcastResp struct {
	Result  bool   `json:"result"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	TxID    string `json:"txid,omitempty"`
}

// GetTransactionInfoResp is the response from gettransactioninfobyid
type GetTransactionInfoResp struct {
	ID          string `json:"id"`
	BlockNumber int64  `json:"blockNumber"`
	Receipt     struct {
		Result string `json:"result"`
	} `json:"receipt"`
}

// TriggerConstantContractResp is the response
type TriggerConstantContractResp struct {
	ConstantResult []string `json:"constant_result"`
	Result         struct {
		Result bool `json:"result"`
	} `json:"result"`
}

// base58ToApiHex converts Tron base58 to API hex (41 + 20 bytes = 42 hex chars)
func base58ToApiHex(base58 string) (string, error) {
	h, err := Base58ToHex(base58)
	if err != nil {
		return "", err
	}
	// Base58ToHex returns 0x + 40 chars (20 bytes). API needs 41 prefix.
	return "41" + strings.TrimPrefix(h, "0x"), nil
}

// CallBalanceOf calls balanceOf(account) on a TRC20 contract
func CallBalanceOf(ctx context.Context, apiURL, tokenBase58, ownerBase58 string) (*big.Int, error) {
	ownerHex, err := base58ToApiHex(ownerBase58)
	if err != nil {
		return nil, err
	}
	tokenHex, err := base58ToApiHex(tokenBase58)
	if err != nil {
		return nil, err
	}
	// balanceOf(address) = 70a08231 + addr padded to 32 bytes
	addr20, _ := hex.DecodeString(strings.TrimPrefix(ownerHex, "41"))
	param := make([]byte, 32)
	copy(param[12:], addr20)
	req := TriggerConstantContractReq{
		OwnerAddress:     ownerHex,
		ContractAddress:  tokenHex,
		FunctionSelector: "balanceOf(address)",
		Parameter:        hex.EncodeToString(param),
	}
	return triggerConstant(ctx, apiURL, req)
}

// CallAllowance calls allowance(owner, spender) on a TRC20 contract
func CallAllowance(ctx context.Context, apiURL, tokenBase58, ownerBase58, spenderBase58 string) (*big.Int, error) {
	ownerHex, err := base58ToApiHex(ownerBase58)
	if err != nil {
		return nil, err
	}
	spenderHex, err := base58ToApiHex(spenderBase58)
	if err != nil {
		return nil, err
	}
	tokenHex, err := base58ToApiHex(tokenBase58)
	if err != nil {
		return nil, err
	}
	// allowance(address,address): pad both to 32 bytes
	owner20, _ := hex.DecodeString(strings.TrimPrefix(ownerHex, "41"))
	spender20, _ := hex.DecodeString(strings.TrimPrefix(spenderHex, "41"))
	param := make([]byte, 64)
	copy(param[12:32], owner20)
	copy(param[44:], spender20)
	req := TriggerConstantContractReq{
		OwnerAddress:     ownerHex,
		ContractAddress:  tokenHex,
		FunctionSelector: "allowance(address,address)",
		Parameter:        hex.EncodeToString(param),
	}
	return triggerConstant(ctx, apiURL, req)
}

// TriggerApprove creates an unsigned approve(spender, amount) transaction
func TriggerApprove(ctx context.Context, apiURL, tokenBase58, ownerBase58, spenderBase58 string, amount *big.Int) (*TronTransaction, error) {
	ownerHex, err := base58ToApiHex(ownerBase58)
	if err != nil {
		return nil, err
	}
	spenderHex, err := base58ToApiHex(spenderBase58)
	if err != nil {
		return nil, err
	}
	tokenHex, err := base58ToApiHex(tokenBase58)
	if err != nil {
		return nil, err
	}
	// approve(address,uint256): spender (32 bytes) + amount (32 bytes)
	spender20, _ := hex.DecodeString(strings.TrimPrefix(spenderHex, "41"))
	param := make([]byte, 64)
	copy(param[12:32], spender20)
	amountBytes := leftPadBytes(amount.Bytes(), 32)
	copy(param[32:], amountBytes)

	req := TriggerSmartContractReq{
		OwnerAddress:     ownerHex,
		ContractAddress:  tokenHex,
		FunctionSelector: "approve(address,uint256)",
		Parameter:        hex.EncodeToString(param),
		FeeLimit:         100_000_000, // 100 TRX in sun
	}
	return triggerSmartContract(ctx, apiURL, req)
}

func triggerSmartContract(ctx context.Context, apiURL string, req TriggerSmartContractReq) (*TronTransaction, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL+"/wallet/triggersmartcontract", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out TriggerSmartContractResp
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("tron triggersmartcontract parse: %w", err)
	}
	tx := out.Transaction
	if tx == nil {
		tx = out.Result
	}
	if tx == nil {
		tx = out.Transaction2
	}
	if tx == nil || tx.RawDataHex == "" {
		return nil, fmt.Errorf("tron triggersmartcontract failed: %s", string(data))
	}
	if out.Code != "" && out.Code != "SUCCESS" {
		return nil, fmt.Errorf("tron triggersmartcontract: %s - %s", out.Code, out.Message)
	}
	return tx, nil
}

// TriggerContractCall creates unsigned tx for contract call. functionSelector is e.g. "transfer(address,uint256)";
// parameterHex is ABI-encoded args (without 4-byte selector).
func TriggerContractCall(ctx context.Context, apiURL, contractBase58, ownerBase58, functionSelector, parameterHex string, feeLimit int64) (*TronTransaction, error) {
	if feeLimit <= 0 {
		feeLimit = 100_000_000
	}
	ownerHex, err := base58ToApiHex(ownerBase58)
	if err != nil {
		return nil, err
	}
	contractHex, err := base58ToApiHex(contractBase58)
	if err != nil {
		return nil, err
	}
	req := TriggerSmartContractReq{
		OwnerAddress:     ownerHex,
		ContractAddress:  contractHex,
		FunctionSelector: functionSelector,
		Parameter:        parameterHex,
		FeeLimit:         feeLimit,
	}
	return triggerSmartContract(ctx, apiURL, req)
}

// SignTransaction signs raw_data_hex with SHA256+ECDSA and adds signature to tx
func SignTransaction(tx *TronTransaction, signHash func([]byte) ([]byte, error)) error {
	rawBytes, err := hex.DecodeString(tx.RawDataHex)
	if err != nil {
		return fmt.Errorf("decode raw_data_hex: %w", err)
	}
	hash := sha256.Sum256(rawBytes)
	sig, err := signHash(hash[:])
	if err != nil {
		return err
	}
	tx.Signature = []string{hex.EncodeToString(sig)}
	return nil
}

// BroadcastTransaction sends signed transaction to network
func BroadcastTransaction(ctx context.Context, apiURL string, tx *TronTransaction) error {
	body, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL+"/wallet/broadcasttransaction", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var out BroadcastResp
	if err := json.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("broadcast parse: %w", err)
	}
	if !out.Result {
		return fmt.Errorf("tron broadcast failed: %s - %s", out.Code, out.Message)
	}
	return nil
}

// GetTransactionInfo fetches transaction status. Returns map with "status" (confirmed/failed) and "blockNumber".
func GetTransactionInfo(ctx context.Context, apiURL, txID string) (map[string]any, error) {
	body := []byte(`{"value":"` + txID + `"}`)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL+"/wallet/gettransactioninfobyid", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out GetTransactionInfoResp
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse tx info: %w", err)
	}
	status := "confirmed"
	if out.Receipt.Result == "FAILED" {
		status = "failed"
	}
	return map[string]any{
		"hash":        txID,
		"blockNumber": fmt.Sprintf("%d", out.BlockNumber),
		"status":      status,
	}, nil
}

func leftPadBytes(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	out := make([]byte, n)
	copy(out[n-len(b):], b)
	return out
}

func triggerConstant(ctx context.Context, apiURL string, req TriggerConstantContractReq) (*big.Int, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL+"/wallet/triggerconstantcontract", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out TriggerConstantContractResp
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if !out.Result.Result || len(out.ConstantResult) == 0 {
		return nil, fmt.Errorf("tron triggerconstantcontract failed: %s", string(data))
	}
	resultHex := out.ConstantResult[0]
	resultBytes, err := hex.DecodeString(resultHex)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(resultBytes), nil
}
