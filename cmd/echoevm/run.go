package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/spf13/cobra"
)

var runFlags struct {
	code     string
	debug    bool
	prestate string // Path to pre-state JSON
	tx       string // Path to transaction JSON
	block    string // Path to block context JSON
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [hex_code]",
		Short: "Run EVM bytecode or execute a transaction with state",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRun,
	}
	cmd.Flags().BoolVar(&runFlags.debug, "debug", false, "Enable debug mode (step-by-step trace)")
	cmd.Flags().StringVar(&runFlags.prestate, "prestate", "", "Path to pre-state JSON file")
	cmd.Flags().StringVar(&runFlags.tx, "tx", "", "Path to transaction JSON file")
	cmd.Flags().StringVar(&runFlags.block, "block", "", "Path to block context JSON file")
	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	// If prestate/tx are provided, we run in "transaction mode"
	if runFlags.prestate != "" || runFlags.tx != "" {
		return runTransactionMode(cmd)
	}

	// Otherwise, run in "simple bytecode mode"
	var codeHex string
	if len(args) > 0 {
		codeHex = args[0]
	} else {
		return fmt.Errorf("provide hex code as argument")
	}

	codeHex = strings.TrimPrefix(codeHex, "0x")
	code, err := hex.DecodeString(codeHex)
	if err != nil {
		return fmt.Errorf("invalid hex code: %w", err)
	}

	statedb := core.NewMemoryStateDB()
	intr := vm.New(code, statedb, common.Address{})

	if runFlags.debug {
		fmt.Printf("%-5s %-15s %-10s %-20s\n", "PC", "OP", "GAS", "STACK (Top)")
		fmt.Println(strings.Repeat("-", 60))

		intr.RunWithHook(func(s vm.TraceStep) bool {
			if s.IsPost { // Post-execution
				stackTop := ""
				if s.StackSize > 0 {
					stackTop = s.Stack[s.StackSize-1]
				}
				fmt.Printf("%04x  %-15s %-10d %s\n", s.PC, s.OpcodeName, 0, stackTop) // Gas not tracked yet
			}
			return true
		})
	} else {
		intr.Run()
	}

	if intr.IsReverted() {
		fmt.Println("Execution Reverted")
	} else {
		ret := intr.ReturnedCode()
		fmt.Printf("Return: 0x%x\n", ret)
	}
	return nil
}

func runTransactionMode(cmd *cobra.Command) error {
	// 1. Load Pre-state
	statedb := core.NewMemoryStateDB()
	if runFlags.prestate != "" {
		data, err := os.ReadFile(runFlags.prestate)
		if err != nil {
			return fmt.Errorf("failed to read prestate: %w", err)
		}
		var pre map[string]struct {
			Balance string            `json:"balance"`
			Code    string            `json:"code"`
			Nonce   string            `json:"nonce"`
			Storage map[string]string `json:"storage"`
		}
		if err := json.Unmarshal(data, &pre); err != nil {
			return fmt.Errorf("failed to unmarshal prestate: %w", err)
		}
		for addrStr, acc := range pre {
			addr := common.HexToAddress(addrStr)
			statedb.CreateAccount(addr)

			if acc.Balance != "" {
				bal, _ := new(big.Int).SetString(strings.TrimPrefix(acc.Balance, "0x"), 16)
				statedb.AddBalance(addr, bal)
			}
			if acc.Nonce != "" {
				nonce, _ := new(big.Int).SetString(strings.TrimPrefix(acc.Nonce, "0x"), 16)
				statedb.SetNonce(addr, nonce.Uint64())
			}
			if acc.Code != "" {
				codeBytes, _ := hex.DecodeString(strings.TrimPrefix(acc.Code, "0x"))
				statedb.SetCode(addr, codeBytes)
			}
			for k, v := range acc.Storage {
				key := common.HexToHash(k)
				val := common.HexToHash(v)
				statedb.SetState(addr, key, val)
			}
		}
	}

	// 2. Load Transaction
	var txData struct {
		To       *string `json:"to"`
		Data     string  `json:"data"`
		Value    string  `json:"value"`
		GasLimit string  `json:"gasLimit"`
		GasPrice string  `json:"gasPrice"`
		Sender   string  `json:"sender"`
		Nonce    string  `json:"nonce"`
	}
	if runFlags.tx != "" {
		data, err := os.ReadFile(runFlags.tx)
		if err != nil {
			return fmt.Errorf("failed to read tx: %w", err)
		}
		if err := json.Unmarshal(data, &txData); err != nil {
			return fmt.Errorf("failed to unmarshal tx: %w", err)
		}
	}

	// 3. Load Block Context
	blockCtx := &vm.BlockContext{
		BlockNumber: big.NewInt(0),
		Timestamp:   0,
		GasLimit:    30000000,
		ChainID:     big.NewInt(1),
		Difficulty:  big.NewInt(0),
	}
	if runFlags.block != "" {
		data, err := os.ReadFile(runFlags.block)
		if err != nil {
			return fmt.Errorf("failed to read block context: %w", err)
		}
		var blk struct {
			Number     string `json:"number"`
			Timestamp  string `json:"timestamp"`
			GasLimit   string `json:"gasLimit"`
			Coinbase   string `json:"coinbase"`
			Difficulty string `json:"difficulty"`
			BaseFee    string `json:"baseFee"`
			Random     string `json:"random"`
		}
		if err := json.Unmarshal(data, &blk); err != nil {
			return fmt.Errorf("failed to unmarshal block context: %w", err)
		}
		if blk.Number != "" {
			blockCtx.BlockNumber, _ = new(big.Int).SetString(strings.TrimPrefix(blk.Number, "0x"), 16)
		}
		if blk.Timestamp != "" {
			ts, _ := new(big.Int).SetString(strings.TrimPrefix(blk.Timestamp, "0x"), 16)
			blockCtx.Timestamp = ts.Uint64()
		}
		if blk.GasLimit != "" {
			gl, _ := new(big.Int).SetString(strings.TrimPrefix(blk.GasLimit, "0x"), 16)
			blockCtx.GasLimit = gl.Uint64()
		}
		if blk.Coinbase != "" {
			blockCtx.Coinbase = common.HexToAddress(blk.Coinbase)
		}
		if blk.Difficulty != "" {
			blockCtx.Difficulty, _ = new(big.Int).SetString(strings.TrimPrefix(blk.Difficulty, "0x"), 16)
		}
		if blk.BaseFee != "" {
			blockCtx.BaseFee, _ = new(big.Int).SetString(strings.TrimPrefix(blk.BaseFee, "0x"), 16)
		}
		if blk.Random != "" {
			blockCtx.Random, _ = new(big.Int).SetString(strings.TrimPrefix(blk.Random, "0x"), 16)
		}
	}

	// Construct VM Transaction
	sender := common.HexToAddress(txData.Sender)
	var to *common.Address
	if txData.To != nil {
		addr := common.HexToAddress(*txData.To)
		to = &addr
	}

	value, _ := new(big.Int).SetString(strings.TrimPrefix(txData.Value, "0x"), 16)
	if value == nil {
		value = big.NewInt(0)
	}

	gasLimit, _ := new(big.Int).SetString(strings.TrimPrefix(txData.GasLimit, "0x"), 16)
	if gasLimit == nil {
		gasLimit = big.NewInt(10000000)
	}

	gasPrice, _ := new(big.Int).SetString(strings.TrimPrefix(txData.GasPrice, "0x"), 16)
	if gasPrice == nil {
		gasPrice = big.NewInt(0)
	}

	dataBytes, _ := hex.DecodeString(strings.TrimPrefix(txData.Data, "0x"))

	nonce, _ := new(big.Int).SetString(strings.TrimPrefix(txData.Nonce, "0x"), 16)
	if nonce == nil {
		nonce = big.NewInt(0)
	}

	var tx *types.Transaction
	if to != nil {
		tx = types.NewTransaction(nonce.Uint64(), *to, value, gasLimit.Uint64(), gasPrice, dataBytes)
	} else {
		tx = types.NewContractCreation(nonce.Uint64(), value, gasLimit.Uint64(), gasPrice, dataBytes)
	}

	// Setup Logger
	if runFlags.debug {
		vm.SetLogger(zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.TraceLevel))
	} else {
		vm.SetLogger(zerolog.New(os.Stderr).Level(zerolog.Disabled))
	}

	ret, _, reverted, err := vm.ApplyTransactionWithContext(statedb, tx, sender, blockCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution Error: %v\n", err)
	}

	// Output result JSON
	result := struct {
		Return   string `json:"return"`
		Reverted bool   `json:"reverted"`
		Error    string `json:"error,omitempty"`
	}{
		Return:   hex.EncodeToString(ret),
		Reverted: reverted,
	}
	if err != nil {
		result.Error = err.Error()
	}

	accountsdump := make(map[string]interface{})

	dumpAccount := func(addr common.Address) {
		if !statedb.Exist(addr) {
			return
		}
		storage := make(map[string]string)
		statedb.ForEachStorage(addr, func(key, value common.Hash) bool {
			storage[key.Hex()] = value.Hex()
			return true
		})

		accountsdump[addr.Hex()] = struct {
			Balance string            `json:"balance"`
			Nonce   string            `json:"nonce"`
			Code    string            `json:"code"`
			Storage map[string]string `json:"storage"`
		}{
			Balance: fmt.Sprintf("0x%x", statedb.GetBalance(addr)),
			Nonce:   fmt.Sprintf("0x%x", statedb.GetNonce(addr)),
			Code:    "0x" + hex.EncodeToString(statedb.GetCode(addr)),
			Storage: storage,
		}
	}

	statedb.ForEachAccount(func(addr common.Address) bool {
		dumpAccount(addr)
		return true
	})

	finalOutput := struct {
		Result    interface{}            `json:"result"`
		PostState map[string]interface{} `json:"postState"`
	}{
		Result:    result,
		PostState: accountsdump,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(finalOutput)
}
