package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/spf13/cobra"
)

var callFlags struct {
	binRuntime string
	artifact   string
	function   string
	args       string
	calldata   string
}

func newCallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Call a deployed (runtime) contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCall(cmd)
		},
		Example: "echoevm call -a ./artifacts/Add.json -f add(uint256,uint256) -A 1,2",
	}
	cmd.Flags().StringVarP(&callFlags.artifact, "artifact", "a", "", "Hardhat artifact JSON path")
	cmd.Flags().StringVarP(&callFlags.binRuntime, "bin-runtime", "r", "", "Raw runtime bytecode (.bin) to execute")
	cmd.Flags().StringVarP(&callFlags.function, "function", "f", "", "Function signature e.g. transfer(address,uint256)")
	cmd.Flags().StringVarP(&callFlags.args, "args", "A", "", "Comma separated function arguments")
	cmd.Flags().StringVarP(&callFlags.calldata, "calldata", "d", "", "Full calldata hex overriding function+args")
	return cmd
}

func runCall(cmd *cobra.Command) error {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	var runtimeHex string
	if callFlags.artifact == "" && callFlags.binRuntime == "" {
		return fmt.Errorf("one of --artifact or --bin-runtime must be provided")
	}

	if callFlags.artifact != "" {
		data, err := os.ReadFile(callFlags.artifact)
		if err != nil {
			return err
		}
		var art struct {
			DeployedBytecode string `json:"deployedBytecode"`
			Bytecode         string `json:"bytecode"`
		}
		if err := json.Unmarshal(data, &art); err != nil {
			return err
		}
		runtimeHex = strings.TrimPrefix(art.DeployedBytecode, "0x")
		if runtimeHex == "" {
			runtimeHex = strings.TrimPrefix(art.Bytecode, "0x")
		}
	} else {
		b, err := os.ReadFile(callFlags.binRuntime)
		if err != nil {
			return err
		}
		runtimeHex = strings.TrimSpace(string(b))
	}

	code, err := hex.DecodeString(runtimeHex)
	if err != nil {
		return fmt.Errorf("invalid runtime bytecode: %w", err)
	}

	var calldata []byte
	if callFlags.calldata != "" {
		calldata, err = hex.DecodeString(strings.TrimPrefix(callFlags.calldata, "0x"))
		if err != nil {
			return fmt.Errorf("invalid calldata: %w", err)
		}
	} else if callFlags.function != "" {
		calldata, err = buildCallData(callFlags.function, callFlags.args)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("provide --calldata or --function + --args")
	}

	i := vm.NewWithCallData(code, calldata)
	i.Run()
	if i.IsReverted() {
		logger.Error().Msg("execution reverted")
		os.Exit(1)
	}
	// Print result (top of stack)
	if i.Stack().Len() > 0 {
		logger.Info().Msgf("Result: %s", i.Stack().PeekSafe(0).String())
	} else {
		logger.Info().Msg("No return value")
	}
	return nil
}

// helper funcs are now in abi_utils.go
