package x402

import (
	"strconv"
	"strings"
)

// ZeroAddress is the Ethereum zero address
const ZeroAddress = "0x0000000000000000000000000000000000000000"

// Network identifiers
const (
	TronMainnet = "tron:mainnet"
	TronShasta  = "tron:shasta"
	TronNile    = "tron:nile"
	EVMMainnet  = "eip155:1"
	EVMSepolia  = "eip155:11155111"
	BSCMainnet  = "eip155:56"
	BSCTestnet  = "eip155:97"
	BaseMainnet = "eip155:8453"
)

// NetworkConfig holds network configuration
var NetworkConfig = struct {
	ChainIDs               map[string]int64
	Permit402Addresses map[string]string
	RPCURLs                map[string]string
}{
	ChainIDs: map[string]int64{
		TronMainnet: 728126428,
		TronShasta:  2494104990,
		TronNile:    3448148188,
		EVMMainnet:  1,
		EVMSepolia:  11155111,
		BSCMainnet:  56,
		BSCTestnet:  97,
		BaseMainnet: 8453,
	},
	Permit402Addresses: map[string]string{
		TronMainnet: "",
		TronShasta:  "",
		TronNile:    "",
		BSCMainnet:  "0x105a6f4613a1d1c17ef35d4d5f053fa2e659a958",
		BSCTestnet:  "",
		BaseMainnet: "",
		EVMMainnet:  "",
	},
	RPCURLs: map[string]string{
		BSCTestnet:  "https://data-seed-prebsc-1-s1.binance.org:8545/",
		BSCMainnet:  "https://bsc-dataseed.binance.org/",
		EVMMainnet:  "https://eth.llamarpc.com",
		EVMSepolia:  "https://rpc.sepolia.org",
		BaseMainnet: "https://mainnet.base.org",
	},
}

// GetRPCURL returns RPC URL for an EVM network
func GetRPCURL(network string) string {
	return NetworkConfig.RPCURLs[network]
}

// GetChainID returns chain ID for network
func GetChainID(network string) (int64, error) {
	if strings.HasPrefix(network, "eip155:") {
		parts := strings.SplitN(network, ":", 2)
		if len(parts) != 2 {
			return 0, NewUnsupportedNetworkError("Invalid EVM network: " + network)
		}
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, NewUnsupportedNetworkError("Invalid EVM network: " + network)
		}
		return id, nil
	}
	if id, ok := NetworkConfig.ChainIDs[network]; ok {
		return id, nil
	}
	return 0, NewUnsupportedNetworkError("Unsupported network: " + network)
}

// GetPermit402Address returns Permit402 contract address for network
func GetPermit402Address(network string) string {
	if addr, ok := NetworkConfig.Permit402Addresses[network]; ok {
		return addr
	}
	if strings.HasPrefix(network, "eip155:") {
		return ZeroAddress
	}
	return "T0000000000000000000000000000000"
}
