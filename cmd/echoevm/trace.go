package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/spf13/cobra"
)

var traceFlags struct {
	binRuntime string
	artifact   string
	calldata   string
	function   string
	args       string
	limit      int
	full       bool
}

func newTraceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "trace",
		Short:   "Execute runtime code and emit per-op trace (JSON)",
		RunE:    func(cmd *cobra.Command, args []string) error { return runTrace(cmd) },
		Example: "echoevm trace -a ./Add.json -f add(uint256,uint256) -A 1,2 --limit 50",
	}
	cmd.Flags().StringVarP(&traceFlags.artifact, "artifact", "a", "", "Hardhat artifact JSON path")
	cmd.Flags().StringVarP(&traceFlags.binRuntime, "bin-runtime", "r", "", "Raw runtime bytecode (.bin)")
	cmd.Flags().StringVarP(&traceFlags.function, "function", "f", "", "Function signature")
	cmd.Flags().StringVarP(&traceFlags.args, "args", "A", "", "Comma separated function arguments")
	cmd.Flags().StringVarP(&traceFlags.calldata, "calldata", "d", "", "Full calldata hex")
	cmd.Flags().IntVar(&traceFlags.limit, "limit", 0, "Maximum number of trace steps (0 = no limit)")
	cmd.Flags().BoolVar(&traceFlags.full, "full", false, "Include post-state after each opcode (pre+post)")
	return cmd
}

func runTrace(cmd *cobra.Command) error {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	var runtimeHex string
	if traceFlags.artifact == "" && traceFlags.binRuntime == "" {
		return fmt.Errorf("one of --artifact or --bin-runtime must be provided")
	}
	if traceFlags.artifact != "" {
		data, err := os.ReadFile(traceFlags.artifact)
		if err != nil {
			return err
		}
		var art struct {
			DeployedBytecode string `json:"deployedBytecode"`
			Bytecode         string `json:"bytecode"`
		}
		if err := json.Unmarshal(data, &art); err != nil {
			return err
		}
		runtimeHex = strings.TrimPrefix(art.DeployedBytecode, "0x")
		if runtimeHex == "" {
			runtimeHex = strings.TrimPrefix(art.Bytecode, "0x")
		}
	} else {
		b, err := os.ReadFile(traceFlags.binRuntime)
		if err != nil {
			return err
		}
		runtimeHex = strings.TrimSpace(string(b))
	}
	code, err := hex.DecodeString(runtimeHex)
	if err != nil {
		return fmt.Errorf("invalid runtime bytecode: %w", err)
	}

	var calldata []byte
	if traceFlags.calldata != "" {
		calldata, err = hex.DecodeString(strings.TrimPrefix(traceFlags.calldata, "0x"))
		if err != nil {
			return fmt.Errorf("invalid calldata: %w", err)
		}
	} else if traceFlags.function != "" {
		calldata, err = buildCallData(traceFlags.function, traceFlags.args)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("provide --calldata or --function + --args")
	}

	intr := vm.NewWithCallData(code, calldata, core.NewMemoryStateDB(), common.Address{})
	enc := json.NewEncoder(cmd.OutOrStdout())
	steps := 0
	type jsonStep struct {
		Type       string        `json:"type"`
		Pre        *vm.TraceStep `json:"pre,omitempty"`
		Post       *vm.TraceStep `json:"post,omitempty"`
		Final      bool          `json:"final,omitempty"`
		Reverted   bool          `json:"reverted,omitempty"`
		ReturnData string        `json:"return_data_hex,omitempty"`
	}
	var lastPre *vm.TraceStep
	intr.RunWithHook(func(s vm.TraceStep) bool {
		// Distinguish pre vs post by comparing PC progression: pre has Halt false and matches previous PC? We invoked pre first always.
		if lastPre == nil || lastPre.PC != s.PC || lastPre.Opcode != s.Opcode || lastPre.Halt { // treat as pre
			cp := s
			lastPre = &cp
			if !traceFlags.full {
				_ = enc.Encode(jsonStep{Type: "step", Pre: &cp})
			}
		} else { // post
			cp := s
			if traceFlags.full {
				_ = enc.Encode(jsonStep{Type: "step", Pre: lastPre, Post: &cp})
			}
			if cp.Halt {
				_ = enc.Encode(jsonStep{Type: "final", Final: true, Reverted: cp.Reverted, ReturnData: fmt.Sprintf("0x%x", intr.ReturnedCode())})
			}
			lastPre = nil
		}
		steps++
		if traceFlags.limit > 0 && steps >= traceFlags.limit {
			logger.Warn().Msg("trace step limit reached")
			return false
		}
		return true
	})
	return nil
}
