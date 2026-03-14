package abi

// Permit402ABI is the minimal ABI for permitTransferFrom(permit, owner, signature)
const Permit402ABI = `[
	{
		"inputs": [
			{
				"name": "permit",
				"type": "tuple",
				"components": [
					{
						"name": "meta",
						"type": "tuple",
						"components": [
							{"name": "ptype", "type": "uint8"},
							{"name": "paymentId", "type": "bytes16"},
							{"name": "nonce", "type": "uint256"},
							{"name": "validAfter", "type": "uint256"},
							{"name": "validBefore", "type": "uint256"}
						]
					},
					{"name": "buyer", "type": "address"},
					{
						"name": "payment",
						"type": "tuple",
						"components": [
							{"name": "payToken", "type": "address"},
							{"name": "payAmount", "type": "uint256"},
							{"name": "payTo", "type": "address"}
						]
					},
					{
						"name": "fee",
						"type": "tuple",
						"components": [
							{"name": "feeTo", "type": "address"},
							{"name": "feeAmount", "type": "uint256"}
						]
					}
				]
			},
			{"name": "owner", "type": "address"},
			{"name": "signature", "type": "bytes"}
		],
		"name": "permitTransferFrom",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`
