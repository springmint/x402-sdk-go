package abi

import (
	"strings"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
)

// ERC20ABI is minimal ERC20 ABI for approve, balanceOf, allowance
const ERC20ABIJSON = `[
	{"name":"balanceOf","type":"function","inputs":[{"name":"account","type":"address"}],"outputs":[{"type":"uint256"}]},
	{"name":"allowance","type":"function","inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"outputs":[{"type":"uint256"}]},
	{"name":"approve","type":"function","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"type":"bool"}]}
]`

// ERC20ABI parsed ABI
var ERC20ABI = func() ethabi.ABI {
	a, err := ethabi.JSON(strings.NewReader(ERC20ABIJSON))
	if err != nil {
		panic("x402: failed to parse ERC20 ABI: " + err.Error())
	}
	return a
}()

// ERC20ApproveMethodID is the 4-byte selector for approve(address,uint256)
const ERC20ApproveMethodID = "095ea7b3"
