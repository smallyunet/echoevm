package vm

import (
	"bytes"
	"fmt"
	"math"
	"math/big"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/params"
	"github.com/smallyunet/echoevm/internal/eth/types"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func pragueFloorDataGas(data []byte) (uint64, error) {
	zeroes := uint64(bytes.Count(data, []byte{0}))
	nonZeroes := uint64(len(data)) - zeroes
	if nonZeroes > math.MaxUint64/params.TxTokenPerNonZeroByte {
		return 0, fmt.Errorf("calldata floor gas overflow")
	}
	tokens := zeroes + nonZeroes*params.TxTokenPerNonZeroByte
	if tokens > (math.MaxUint64-params.TxGas)/params.TxCostFloorPerToken {
		return 0, fmt.Errorf("calldata floor gas overflow")
	}
	return params.TxGas + tokens*params.TxCostFloorPerToken, nil
}

func resolveCodeForRules(statedb core.StateDB, address common.Address, rules core.Rules) []byte {
	code := statedb.GetCode(address)
	if !rules.IsPrague {
		return code
	}
	if target, ok := types.ParseDelegation(code); ok {
		return statedb.GetCode(target)
	}
	return code
}

// applySetCodeAuthorizations applies every valid EIP-7702 authorization in
// order. Invalid tuples are ignored after warming their recovered authority,
// matching the transaction-level semantics rather than invalidating the tx.
func applySetCodeAuthorizations(statedb core.StateDB, auths []types.SetCodeAuthorization, rules core.Rules, chainID *big.Int) {
	if !rules.IsPrague || len(auths) == 0 {
		return
	}
	if chainID == nil {
		chainID = rules.ChainID
	}
	for index := range auths {
		auth := &auths[index]
		if !auth.ChainID.IsZero() && auth.ChainID.CmpBig(chainID) != 0 {
			continue
		}
		if auth.Nonce == math.MaxUint64 {
			continue
		}
		authority, err := auth.Authority()
		if err != nil {
			continue
		}
		statedb.AddAddressToAccessList(authority)
		code := statedb.GetCode(authority)
		_, delegated := types.ParseDelegation(code)
		if len(code) != 0 && !delegated {
			continue
		}
		if statedb.GetNonce(authority) != auth.Nonce {
			continue
		}
		if statedb.Exist(authority) {
			statedb.AddRefund(params.CallNewAccountGas - params.TxAuthTupleGas)
		}
		statedb.SetNonce(authority, auth.Nonce+1)
		if auth.Address == (common.Address{}) {
			statedb.SetCode(authority, nil)
		} else {
			statedb.SetCode(authority, types.AddressToDelegation(auth.Address))
		}
	}
}
