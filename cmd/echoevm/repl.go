package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/spf13/cobra"
)

func newReplCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Interactive EVM shell",
		RunE:  runRepl,
	}
}

func runRepl(cmd *cobra.Command, args []string) error {
	fmt.Println("EchoEVM REPL")
	fmt.Println("Type opcodes (e.g., 'PUSH1 01 ADD') or hex (e.g., '600101'). Type 'exit' to quit.")

	// Persistent state
	stack := core.NewStack()
	memory := core.NewMemory()
	statedb := core.NewMemoryStateDB()
	address := common.Address{}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "exit" || line == "quit" {
			break
		}
		if line == "" {
			continue
		}

		// Parse input
		code, err := parseInput(line)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Execute
		intr := vm.New(code, statedb, address)
		intr.SetStack(stack)
		intr.SetMemory(memory)

		// Run
		intr.Run()
		if err := intr.Err(); err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		// Print state
		printState(stack, memory)
	}
	return nil
}

func parseInput(input string) ([]byte, error) {
	// Try hex first if it looks like hex and has no spaces
	if !strings.Contains(input, " ") && len(input)%2 == 0 {
		if code, err := hex.DecodeString(strings.TrimPrefix(input, "0x")); err == nil {
			return code, nil
		}
	}

	// Parse mnemonics
	var code []byte
	parts := strings.Fields(input)
	for i := 0; i < len(parts); i++ {
		part := strings.ToUpper(parts[i])
		op, ok := core.OpcodeByName(part)
		if !ok {
			// Maybe it's a hex value for PUSH?
			// If previous was PUSHx, this might be the data.
			// But here we just parse opcodes.
			// If user types PUSH1 10, we expect PUSH1 opcode then 0x10 byte.

			// Check if it's a hex number
			if val, err := hex.DecodeString(part); err == nil {
				code = append(code, val...)
				continue
			}
			// Check if it's a decimal number?
			// For simplicity, assume hex bytes for data without 0x prefix if length is even,
			// or just error out.
			return nil, fmt.Errorf("unknown opcode or invalid data: %s", part)
		}
		code = append(code, op)
	}
	return code, nil
}

func printState(stack *core.Stack, memory *core.Memory) {
	// Print Stack
	st := stack.Snapshot()
	fmt.Printf("Stack [%d]:\n", len(st))
	for i := len(st) - 1; i >= 0; i-- {
		fmt.Printf("  %04d: %s\n", i, st[i])
	}

	// Print Memory (only if not empty)
	if memory.Len() > 0 {
		fmt.Printf("Memory [%d bytes]:\n", memory.Len())
		data := memory.Data()
		// Print in 32-byte chunks
		for i := 0; i < len(data); i += 32 {
			end := i + 32
			if end > len(data) {
				end = len(data)
			}
			fmt.Printf("  %04x: %x\n", i, data[i:end])
		}
	}
}
