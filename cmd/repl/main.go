//go:build evmrepl

package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/smallyunet/echoevm/utils"
)

func main() {
	bin := flag.String("bin", "", "path to contract .bin file (required)")
	flag.Parse()
	if *bin == "" {
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*bin)
	if err != nil {
		panic(err)
	}
	code, err := hex.DecodeString(string(data))
	if err != nil {
		panic(err)
	}

	cw := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Kitchen}
	logger := zerolog.New(cw).With().Timestamp().Logger()
	zerologlog.Logger = logger
	vm.SetLogger(logger)

	utils.PrintBytecode(logger, code, zerolog.InfoLevel)

	interp := vm.New(code)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Type ENTER to execute next opcode, 'quit' to exit")
	for {
		fmt.Print("repl> ")
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(line) == "quit" {
			break
		}
		op, cont := interp.Step()
		fmt.Printf("pc: 0x%04x op: %s\n", interp.PC(), core.OpcodeName(op))
		fmt.Printf("stack: %v\n", interp.Stack().Snapshot())
		fmt.Printf("memory: %s\n", interp.Memory().Snapshot())
		if !cont {
			fmt.Println("execution finished")
			break
		}
	}
}
