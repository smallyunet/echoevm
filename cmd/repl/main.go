//go:build evmrepl

package main

import (
	"encoding/hex"
	"flag"
	"os"
	"time"

	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
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

	r := NewREPL(code, os.Stdout, os.Stdin)
	r.Run()
}
