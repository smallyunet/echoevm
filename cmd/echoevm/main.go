package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/smallyunet/echoevm/utils"
)

func main() {
	// --- Step 1: Read hex-encoded constructor bytecode from file ---
	data, err := os.ReadFile("build/Add.bin")
	check(err, "failed to read bytecode file")

	// --- Step 2: Decode hex string to bytecode []byte ---
	code, err := hex.DecodeString(string(data))
	check(err, "failed to decode hex bytecode")

	// --- Step 3: Optional debug output ---
	fmt.Println("=== Disassembled Bytecode ===")
	utils.PrintBytecode(code)

	// --- Step 4: Create and run the interpreter with constructor bytecode ---
	interpreter := vm.New(code)
	interpreter.Run()

	// --- Step 5: Inspect stack state ---
	switch interpreter.Stack().Len() {
	case 1:
		fmt.Printf("Final Result on Stack: %s\n", interpreter.Stack().Peek(0).String())
	case 0:
		fmt.Println("Execution finished. Stack is empty.")
	default:
		fmt.Printf("Execution finished. Stack height = %d\n", interpreter.Stack().Len())
	}
}

// check is a helper to panic with context on error
func check(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}
