//go:build evmrepl

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

// REPL drives an interpreter interactively.
type REPL struct {
	interp *vm.Interpreter
	in     *bufio.Reader
	out    io.Writer
}

// NewREPL creates a REPL around the given bytecode.
func NewREPL(code []byte, out io.Writer, in io.Reader) *REPL {
	r := &REPL{
		interp: vm.New(code),
		out:    out,
		in:     bufio.NewReader(in),
	}
	return r
}

// Run starts the interactive loop.
func (r *REPL) Run() {
	fmt.Fprintln(r.out, "Commands: <enter>|step, stack, mem, quit")
	for {
		fmt.Fprint(r.out, "repl> ")
		line, _ := r.in.ReadString('\n')
		switch strings.TrimSpace(line) {
		case "", "step", "n":
			op, cont := r.interp.Step()
			fmt.Fprintf(r.out, "pc: 0x%04x op: %s\n", r.interp.PC(), core.OpcodeName(op))
			fmt.Fprintf(r.out, "stack: %v\n", r.interp.Stack().Snapshot())
			fmt.Fprintf(r.out, "memory: %s\n", r.interp.Memory().Snapshot())
			if !cont {
				fmt.Fprintln(r.out, "execution finished")
				return
			}
		case "stack":
			fmt.Fprintf(r.out, "stack: %v\n", r.interp.Stack().Snapshot())
		case "mem", "memory":
			fmt.Fprintf(r.out, "memory: %s\n", r.interp.Memory().Snapshot())
		case "quit", "exit":
			return
		default:
			fmt.Fprintln(r.out, "unknown command")
		}
	}
}
