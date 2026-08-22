package vm

import (
	"fmt"
	"math/big"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/types"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func (i *Interpreter) rules() core.Rules {
	config := i.chainConfig
	if config == nil {
		config = core.DefaultChainConfig
	}
	return config.Rules(new(big.Int).SetUint64(i.blockNumber))
}

// resolveCode follows exactly one EIP-7702 delegation designator after Prague.
// EXTCODE* opcodes intentionally continue to expose the designator itself; this
// helper is only for code selected for CALL-like execution.
func (i *Interpreter) resolveCode(address common.Address) []byte {
	code := i.statedb.GetCode(address)
	if !i.rules().IsPrague {
		return code
	}
	if target, ok := types.ParseDelegation(code); ok {
		return i.statedb.GetCode(target)
	}
	return code
}

// chargeDelegationResolution applies EIP-7702's warm/cold account access cost
// for the delegation target selected by a CALL-like opcode.
func (i *Interpreter) chargeDelegationResolution(address common.Address) bool {
	if !i.rules().IsPrague {
		return true
	}
	target, ok := types.ParseDelegation(i.statedb.GetCode(address))
	if !ok {
		return true
	}
	cost := uint64(core.GasWarmStorageRead)
	if !i.statedb.AddressInAccessList(target) {
		cost = core.GasColdAccountAccess
		i.statedb.AddAddressToAccessList(target)
	}
	if i.gas < cost {
		i.err = fmt.Errorf("out of gas: EIP-7702 delegation access")
		i.reverted = true
		return false
	}
	i.gas -= cost
	return true
}

func (i *Interpreter) runCallPrecompile(address common.Address, input []byte, returnOffset, returnLength, gasLimit uint64, snapshot int) bool {
	rules := i.rules()
	if !IsPrecompiledForRules(address, rules) {
		return false
	}
	ret, remainingGas, err := RunPrecompiledForRules(address, input, gasLimit, rules)
	i.returnData = ret
	if err != nil {
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(new(big.Int))
		return true
	}
	i.gas += remainingGas
	toCopy := uint64(len(ret))
	if toCopy > returnLength {
		toCopy = returnLength
	}
	if toCopy > 0 {
		i.memory.Write(returnOffset, ret[:toCopy])
	}
	i.stack.PushSafe(big.NewInt(1))
	return true
}
