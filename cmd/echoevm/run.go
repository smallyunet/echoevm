package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/spf13/cobra"
)

var runFlags struct {
	code  string
	debug bool
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [hex_code]",
		Short: "Run EVM bytecode",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRun,
	}
	cmd.Flags().BoolVar(&runFlags.debug, "debug", false, "Enable debug mode (step-by-step trace)")
	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	var codeHex string
	if len(args) > 0 {
		codeHex = args[0]
	} else {
		// Read from stdin? Or just error for now.
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
