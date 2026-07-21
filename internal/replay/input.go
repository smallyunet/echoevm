package replay

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func ParseTransactionReference(input string) (transactionReference, error) {
	value := strings.TrimSpace(input)
	if common.IsHexHash(value) {
		return transactionReference{Hash: common.HexToHash(value)}, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return transactionReference{}, fmt.Errorf("enter a 32-byte transaction hash or an Etherscan transaction URL")
	}
	host := strings.ToLower(parsed.Hostname())
	chainID := uint64(0)
	switch host {
	case "etherscan.io", "www.etherscan.io":
		chainID = 1
	case "sepolia.etherscan.io":
		chainID = 11155111
	default:
		return transactionReference{}, fmt.Errorf("unsupported explorer host %q; use etherscan.io or sepolia.etherscan.io", host)
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 2 || parts[0] != "tx" {
		return transactionReference{}, fmt.Errorf("etherscan URL must use /tx/<transaction-hash>")
	}
	hashValue, err := url.PathUnescape(parts[1])
	if err != nil || !common.IsHexHash(hashValue) {
		return transactionReference{}, fmt.Errorf("etherscan URL contains an invalid transaction hash")
	}
	return transactionReference{Hash: common.HexToHash(hashValue), ChainID: chainID}, nil
}
