package main

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
)

// buildCallData creates ABI encoded calldata from a function signature and
// comma-separated arguments. Only a few basic types (uint256,int256,bool,string)
// are supported. Numeric values can be provided in decimal or 0x-prefixed hex.
func buildCallData(sig, argString string) ([]byte, error) {
	open := strings.Index(sig, "(")
	close := strings.LastIndex(sig, ")")
	if open == -1 || close == -1 || close < open {
		return nil, fmt.Errorf("invalid function signature")
	}
	typesPart := sig[open+1 : close]
	typeNames := []string{}
	if len(typesPart) > 0 {
		for _, t := range strings.Split(typesPart, ",") {
			typeNames = append(typeNames, strings.TrimSpace(t))
		}
	}

	args := []string{}
	if len(argString) > 0 {
		for _, a := range strings.Split(argString, ",") {
			args = append(args, strings.TrimSpace(a))
		}
	}

	if len(typeNames) != len(args) {
		return nil, fmt.Errorf("argument count mismatch")
	}

	var abiArgs abi.Arguments
	values := make([]interface{}, len(args))
	for i, tname := range typeNames {
		at, err := abi.NewType(tname, "", nil)
		if err != nil {
			return nil, err
		}
		abiArgs = append(abiArgs, abi.Argument{Type: at})
		val, err := parseArg(args[i], at)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	encoded, err := abiArgs.Pack(values...)
	if err != nil {
		return nil, err
	}
	selector := crypto.Keccak256([]byte(sig))[:4]
	return append(selector, encoded...), nil
}

func parseArg(val string, typ abi.Type) (interface{}, error) {
	switch typ.T {
	case abi.UintTy, abi.IntTy:
		n := new(big.Int)
		var ok bool
		if strings.HasPrefix(val, "0x") {
			n, ok = n.SetString(val[2:], 16)
		} else {
			n, ok = n.SetString(val, 10)
		}
		if !ok {
			return nil, fmt.Errorf("invalid integer value: %s", val)
		}
		return n, nil
	case abi.BoolTy:
		return strings.ToLower(val) == "true", nil
	case abi.StringTy:
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typ.String())
	}
}
