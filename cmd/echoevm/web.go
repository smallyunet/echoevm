package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/smallyunet/echoevm/internal/config"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	webui "github.com/smallyunet/echoevm/internal/web"
	"github.com/spf13/cobra"
)

type webFlags struct {
	addr string
	code string
}

type webMessage struct {
	Type       string        `json:"type"`
	Pre        *vm.TraceStep `json:"pre,omitempty"`
	Post       *vm.TraceStep `json:"post,omitempty"`
	MemoryHex  string        `json:"memory_hex,omitempty"`
	Reverted   bool          `json:"reverted,omitempty"`
	ReturnData string        `json:"return_data_hex,omitempty"`
	Error      string        `json:"error,omitempty"`
}

func newWebCmd() *cobra.Command {
	flags := &webFlags{}
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start the browser-based EVM debugger",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWeb(flags)
		},
	}
	cmd.Flags().StringVar(&flags.addr, "addr", ":8080", "HTTP listen address")
	cmd.Flags().StringVar(&flags.code, "code", "", "EVM bytecode as a hex string")
	_ = cmd.MarkFlagRequired("code")
	return cmd
}

func runWeb(flags *webFlags) error {
	code, err := hex.DecodeString(strings.TrimPrefix(flags.code, "0x"))
	if err != nil {
		return fmt.Errorf("invalid bytecode: %w", err)
	}
	if len(code) == 0 {
		return fmt.Errorf("bytecode must not be empty")
	}

	server := webui.NewServer(flags.addr)
	go serveWebRuns(server, code)
	return server.Start()
}

func serveWebRuns(server *webui.Server, code []byte) {
	for control := range server.Control() {
		if control.Type != "run" {
			log.Warn().Str("type", control.Type).Msg("Ignoring unsupported web control message")
			continue
		}

		broadcastWebMessage(server, webMessage{Type: "start"})
		intr := vm.New(code, core.NewMemoryStateDB(), common.Address{})
		intr.SetGas(config.DefaultGasLimit)
		intr.RunWithHook(func(step vm.TraceStep) bool {
			stepCopy := step
			message := webMessage{Type: "step"}
			if step.IsPost {
				message.Post = &stepCopy
				message.MemoryHex = hex.EncodeToString(intr.Memory().Data())
			} else {
				message.Pre = &stepCopy
			}
			broadcastWebMessage(server, message)
			return true
		})

		final := webMessage{
			Type:       "final",
			Reverted:   intr.IsReverted(),
			ReturnData: fmt.Sprintf("0x%x", intr.ReturnedCode()),
		}
		if intr.Err() != nil {
			final.Error = intr.Err().Error()
		}
		broadcastWebMessage(server, final)
	}
}

func broadcastWebMessage(server *webui.Server, message webMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode web debugger message")
		return
	}
	server.Broadcast(data)
}
