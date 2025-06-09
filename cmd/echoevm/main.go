package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/smallyunet/echoevm/utils"
)

func main() {
	// Command line flags
	binPath := flag.String("bin", "build/Add.bin", "path to contract .bin file")
	mode := flag.String("mode", "full", "execution mode: deploy or full")
	flag.Parse()

	// --- Step 1: Read hex-encoded constructor bytecode from file ---
	data, err := os.ReadFile(*binPath)
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

	// --- Step 5: Inspect stack state after constructor execution ---
	switch interpreter.Stack().Len() {
	case 1:
		fmt.Printf("Final Result on Stack: %s\n", interpreter.Stack().Peek(0).String())
	case 0:
		fmt.Println("Execution finished. Stack is empty.")
	default:
		fmt.Printf("Execution finished. Stack height = %d\n", interpreter.Stack().Len())
	}

	// --- Step 6: If constructor returned runtime code and mode is "full", execute it ---
	runtimeCode := interpreter.ReturnedCode()
	if *mode == "full" && len(runtimeCode) > 0 {
		fmt.Println("=== Runtime Bytecode ===")
		utils.PrintBytecode(runtimeCode)

		// Example calldata for Add.add(uint256,uint256) with arguments
		// 1 and 2. This is the ABI-encoded function selector and
		// parameters.
		callData, _ := hex.DecodeString(
			"771602f700000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		)
		runtimeInterpreter := vm.NewWithCallData(runtimeCode, callData)
		runtimeInterpreter.Run()

		switch runtimeInterpreter.Stack().Len() {
		case 1:
			fmt.Printf("Runtime Result on Stack: %s\n", runtimeInterpreter.Stack().Peek(0).String())
		case 0:
			fmt.Println("Runtime execution finished. Stack is empty.")
		default:
			fmt.Printf("Runtime execution finished. Stack height = %d\n", runtimeInterpreter.Stack().Len())
		}
	}
}

// check is a helper to panic with context on error
func check(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}
