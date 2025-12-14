package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/spf13/cobra"
)

type disasmFlags struct {
	binFile      string
	artifactFile string
	runtime      bool
}

func newDisasmCmd() *cobra.Command {
	flags := &disasmFlags{}
	cmd := &cobra.Command{
		Use:   "disasm [hex]",
		Short: "Disassemble EVM bytecode into human-readable opcodes",
		Long: `Disassemble EVM bytecode from:
- A hex string argument
- A .bin file (--bin)
- A Hardhat artifact JSON (--artifact)

By default uses constructor bytecode from artifacts. Use --runtime to disassemble deployedBytecode.`,
		Example: `  echoevm disasm 6001600201
  echoevm disasm -b ./contract.bin
  echoevm disasm -a ./artifacts/Add.json
  echoevm disasm -a ./artifacts/Add.json --runtime`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisasm(cmd, args, flags)
		},
	}

	cmd.Flags().StringVarP(&flags.binFile, "bin", "b", "", "Path to .bin file containing bytecode")
	cmd.Flags().StringVarP(&flags.artifactFile, "artifact", "a", "", "Path to Hardhat artifact JSON")
	cmd.Flags().BoolVarP(&flags.runtime, "runtime", "r", false, "Use deployedBytecode from artifact (default: constructor bytecode)")

	return cmd
}

func runDisasm(cmd *cobra.Command, args []string, flags *disasmFlags) error {
	var bytecode []byte
	var err error

	switch {
	case flags.artifactFile != "":
		bytecode, err = loadBytecodeFromArtifact(flags.artifactFile, flags.runtime)
	case flags.binFile != "":
		bytecode, err = loadBytecodeFromBinFile(flags.binFile)
	case len(args) > 0:
		bytecode, err = decodeBytecodeHex(args[0])
	default:
		return fmt.Errorf("provide bytecode as argument, --bin file, or --artifact file")
	}

	if err != nil {
		return fmt.Errorf("failed to load bytecode: %w", err)
	}

	if len(bytecode) == 0 {
		return fmt.Errorf("empty bytecode")
	}

	// Disassemble and output
	instructions := disassemble(bytecode)

	if globalFlags.output == "json" {
		return outputDisasmJSON(cmd, instructions)
	}
	return outputDisasmPlain(cmd, instructions)
}

// Instruction represents a single disassembled instruction
type Instruction struct {
	Offset     uint64 `json:"offset"`
	OpcodeByte byte   `json:"opcode_byte"`
	OpcodeName string `json:"opcode_name"`
	Operand    string `json:"operand,omitempty"`
	RawBytes   string `json:"raw_bytes"`
}

func disassemble(code []byte) []Instruction {
	var instructions []Instruction
	pc := uint64(0)

	for pc < uint64(len(code)) {
		op := code[pc]
		opName := core.OpcodeName(op)
		inst := Instruction{
			Offset:     pc,
			OpcodeByte: op,
			OpcodeName: opName,
		}

		// Check if it's a PUSH instruction
		if op >= core.PUSH1 && op <= 0x7f {
			pushSize := int(op - core.PUSH1 + 1)
			startPC := pc

			// Extract operand bytes
			operandStart := pc + 1
			operandEnd := operandStart + uint64(pushSize)

			if operandEnd > uint64(len(code)) {
				// Truncated PUSH - handle gracefully
				operandEnd = uint64(len(code))
			}

			operandBytes := code[operandStart:operandEnd]
			inst.Operand = hex.EncodeToString(operandBytes)
			inst.RawBytes = hex.EncodeToString(code[startPC:operandEnd])

			pc = operandEnd
		} else {
			inst.RawBytes = hex.EncodeToString([]byte{op})
			pc++
		}

		instructions = append(instructions, inst)
	}

	return instructions
}

func outputDisasmPlain(cmd *cobra.Command, instructions []Instruction) error {
	for _, inst := range instructions {
		if inst.Operand != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "%04x: %s %s\n", inst.Offset, inst.OpcodeName, inst.Operand)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%04x: %s\n", inst.Offset, inst.OpcodeName)
		}
	}
	return nil
}

func outputDisasmJSON(cmd *cobra.Command, instructions []Instruction) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(instructions)
}

// Helper functions for loading bytecode

func loadBytecodeFromArtifact(path string, useRuntime bool) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var artifact struct {
		Bytecode         string `json:"bytecode"`
		DeployedBytecode string `json:"deployedBytecode"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("invalid artifact JSON: %w", err)
	}

	hexStr := artifact.Bytecode
	if useRuntime {
		hexStr = artifact.DeployedBytecode
		if hexStr == "" {
			return nil, fmt.Errorf("artifact has no deployedBytecode")
		}
	}

	if hexStr == "" {
		return nil, fmt.Errorf("artifact has no bytecode")
	}

	return decodeBytecodeHex(hexStr)
}

func loadBytecodeFromBinFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	hexStr := strings.TrimSpace(string(data))
	return decodeBytecodeHex(hexStr)
}

func decodeBytecodeHex(hexStr string) ([]byte, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	hexStr = strings.TrimSpace(hexStr)
	return hex.DecodeString(hexStr)
}
