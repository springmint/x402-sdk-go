# x402-sdk-go

[![Go](https://img.shields.io/badge/Go-1.24-blue)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Go SDK for the **x402** payment protocol. Implements the full HTTP 402 Payment Required flow — client signing, server paywall middleware, and facilitator settlement.

---

## What is x402?

**[x402](https://www.x402.org/)** is an open standard for internet-native payments using the HTTP 402 status code. This SDK provides the Go implementation for integrating x402 into your services.

---

## Architecture

The SDK has three roles:

| Role | Description | Key Files |
|------|-------------|-----------|
| **Client** | Signs EIP-712 payment permits, handles 402 retry | `client.go`, `http_client.go` |
| **Server** | Protects endpoints with paywall middleware, returns 402 | `server.go`, `middleware/` |
| **Facilitator** | Verifies signatures, settles on-chain via Permit402 contract | `facilitator.go`, `facilitator_client.go` |

### Payment Flow

```
Client Request → Server returns 402 + PaymentRequired
       ↓
Client signs EIP-712 permit → Retries with PAYMENT-SIGNATURE header
       ↓
Server → Facilitator verifies signature + calls Permit402.permitTransferFrom()
       ↓
Settlement confirmed → Server returns resource
```

---

## Installation

```bash
go get github.com/springmint/x402-sdk-go
```

---

## Quick Start

### Server (Paywall Middleware)

```go
import (
    x402 "github.com/springmint/x402-sdk-go"
    "github.com/springmint/x402-sdk-go/middleware"
)

// Gin example
router.Use(ginmw.PaywallMiddleware(middleware.PaywallConfig{
    Scheme:          "permit402",
    Network:         x402.BSCMainnet,
    PayToAddress:    "0xYourAddress",
    Amount:          "1000000", // 1 USDT (6 decimals)
    Asset:           "USDT",
    FacilitatorURL:  "https://cppay.finance/api/x402/facilitator",
}))
```

### Client (Auto 402 Handling)

```go
import x402 "github.com/springmint/x402-sdk-go"

client := x402.NewX402HTTPClient(privateKey, rpcURL)
resp, err := client.Get("https://api.example.com/protected-resource")
```

---

## Supported Networks

| Network | Identifier | Permit402 Address |
|---------|-----------|-------------------|
| TRON Mainnet | `tron:mainnet` | `TT8rEWbCoNX7vpEUauxb7rWJsTgs8vDLAn` |
| TRON Nile | `tron:nile` | `TFxDcGvS7zfQrS1YzcCMp673ta2NHHzsiH` |
| TRON Shasta | `tron:shasta` | `TR2XninQ3jsvRRLGTifFyUHTBysffooUjt` |
| BSC Mainnet | `eip155:56` | `0x105a6f4613a1d1c17ef35d4d5f053fa2e659a958` |

## Supported Tokens

- **USDT** — TRON, BSC
- **USDC** — BSC, Ethereum, Base

---

## Project Structure

```
├── abi/              # Contract ABI definitions (Permit402, ERC-20, EIP-712)
├── mechanisms/       # Payment scheme implementations (permit402, transfer_auth)
├── middleware/       # HTTP paywall middleware (net/http, Gin, GoFrame)
├── signers/          # EIP-712 signing and verification
├── tokens/           # Token registry (symbol → address per chain)
├── utils/            # EIP-712 hashing, provider resolution
├── setup/            # Convenience setup helpers
├── client.go         # X402Client — client-side payment orchestration
├── server.go         # X402Server — server-side payment validation
├── facilitator.go    # FacilitatorClient interface
├── config.go         # Network configs (chain IDs, contract addresses, RPCs)
├── types.go          # Core data structures
└── http_client.go    # HTTP client with automatic 402 retry
```

---

## License

[MIT](LICENSE)
