package abi

// Permit402PrimaryType is the EIP-712 primary type (matches contract Permit402Details)
const Permit402PrimaryType = "Permit402Details"

// GetPermit402EIP712Types returns EIP-712 type definitions matching IPermit402
func GetPermit402EIP712Types() map[string][]map[string]string {
	return map[string][]map[string]string{
		"EIP712Domain": {
			{"name": "name", "type": "string"},
			{"name": "chainId", "type": "uint256"},
			{"name": "verifyingContract", "type": "address"},
		},
		"PermitMeta": {
			{"name": "ptype", "type": "uint8"},
			{"name": "paymentId", "type": "bytes16"},
			{"name": "nonce", "type": "uint256"},
			{"name": "validAfter", "type": "uint256"},
			{"name": "validBefore", "type": "uint256"},
		},
		"Payment": {
			{"name": "payToken", "type": "address"},
			{"name": "payAmount", "type": "uint256"},
			{"name": "payTo", "type": "address"},
		},
		"Fee": {
			{"name": "feeTo", "type": "address"},
			{"name": "feeAmount", "type": "uint256"},
		},
		"Permit402Details": {
			{"name": "meta", "type": "PermitMeta"},
			{"name": "buyer", "type": "address"},
			{"name": "payment", "type": "Payment"},
			{"name": "fee", "type": "Fee"},
		},
	}
}
