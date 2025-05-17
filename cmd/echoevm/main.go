package main

import (
	"fmt"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

func main() {
	// 0x60 0x05 0x60 0x06 0x01 0x00
	// PUSH1 5, PUSH1 6, ADD, STOP
	code := []byte{0x60, 0x05, 0x60, 0x06, 0x01, 0x00}

	interpreter := vm.New(code)
	interpreter.Run()

	if interpreter.Stack().Len() != 1 {
		panic("unexpected stack height")
	}
	fmt.Printf("Result: %d\n", interpreter.Stack().Pop()) // should print 11
}
