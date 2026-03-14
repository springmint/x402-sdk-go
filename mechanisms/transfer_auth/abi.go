package transfer_auth

// TransferWithAuthorizationABI is the minimal ABI for transferWithAuthorization(from,to,value,validAfter,validBefore,nonce,v,r,s)
const TransferWithAuthorizationABI = `[
	{
		"inputs": [
			{"name": "from", "type": "address"},
			{"name": "to", "type": "address"},
			{"name": "value", "type": "uint256"},
			{"name": "validAfter", "type": "uint256"},
			{"name": "validBefore", "type": "uint256"},
			{"name": "nonce", "type": "bytes32"},
			{"name": "v", "type": "uint8"},
			{"name": "r", "type": "bytes32"},
			{"name": "s", "type": "bytes32"}
		],
		"name": "transferWithAuthorization",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`
