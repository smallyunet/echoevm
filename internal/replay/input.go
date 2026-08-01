package replay

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const ethereumMainnetChainID uint64 = 1

func ParseTransactionReference(input string) (transactionReference, error) {
	value := strings.TrimSpace(input)
	if common.IsHexHash(value) {
		return transactionReference{Hash: common.HexToHash(value), ChainID: ethereumMainnetChainID}, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return transactionReference{}, fmt.Errorf("enter a 32-byte transaction hash or an Etherscan transaction URL")
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "etherscan.io", "www.etherscan.io":
	default:
		return transactionReference{}, fmt.Errorf("unsupported explorer host %q; only Ethereum Mainnet etherscan.io URLs are accepted", host)
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 2 || parts[0] != "tx" {
		return transactionReference{}, fmt.Errorf("etherscan URL must use /tx/<transaction-hash>")
	}
	hashValue, err := url.PathUnescape(parts[1])
	if err != nil || !common.IsHexHash(hashValue) {
		return transactionReference{}, fmt.Errorf("etherscan URL contains an invalid transaction hash")
	}
	return transactionReference{Hash: common.HexToHash(hashValue), ChainID: ethereumMainnetChainID}, nil
}
