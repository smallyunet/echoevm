package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/spf13/cobra"
)

var deployFlags struct {
	binPath   string
	artifact  string
	out       string
	printCode bool
}

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy constructor bytecode and obtain runtime code",
		RunE:    func(cmd *cobra.Command, args []string) error { return runDeploy(cmd) },
		Example: "echoevm deploy -b ./Test.bin --out runtime.bin",
	}
	cmd.Flags().StringVarP(&deployFlags.binPath, "bin", "b", "", "Constructor .bin file path")
	cmd.Flags().StringVarP(&deployFlags.artifact, "artifact", "a", "", "Hardhat artifact JSON path")
	// Use --out-file to avoid clashing with global --output (-o) flag
	cmd.Flags().StringVar(&deployFlags.out, "out-file", "", "Write runtime bytecode to file")
	cmd.Flags().BoolVar(&deployFlags.printCode, "print", false, "Print runtime hex to stdout")
	return cmd
}

func runDeploy(cmd *cobra.Command) error {
	if deployFlags.binPath == "" && deployFlags.artifact == "" {
		return errors.New("provide --bin or --artifact")
	}
	logger := zerolog.New(cmd.OutOrStdout()).With().Timestamp().Logger()

	var constructorHex string
	if deployFlags.artifact != "" {
		data, err := os.ReadFile(deployFlags.artifact)
		if err != nil {
			return err
		}
		var art struct {
			Bytecode         string `json:"bytecode"`
			DeployedBytecode string `json:"deployedBytecode"`
		}
		if err := json.Unmarshal(data, &art); err != nil {
			return err
		}
		constructorHex = strings.TrimPrefix(art.Bytecode, "0x")
		if constructorHex == "" {
			return fmt.Errorf("artifact missing bytecode field")
		}
	} else {
		b, err := os.ReadFile(deployFlags.binPath)
		if err != nil {
			return err
		}
		constructorHex = strings.TrimSpace(string(b))
	}

	code, err := hex.DecodeString(constructorHex)
	if err != nil {
		return fmt.Errorf("invalid constructor bytecode: %w", err)
	}

	intr := vm.New(code)
	intr.Run()
	runtime := intr.ReturnedCode()
	if len(runtime) == 0 {
		logger.Warn().Msg("No runtime code returned (empty RETURN range)")
	}
	runtimeHex := hex.EncodeToString(runtime)

	if deployFlags.out != "" {
		if err := os.WriteFile(deployFlags.out, []byte(runtimeHex), 0o644); err != nil {
			return fmt.Errorf("failed writing runtime code: %w", err)
		}
		logger.Info().Msgf("Runtime code written to %s (%d bytes)", deployFlags.out, len(runtime))
	}
	if deployFlags.printCode || deployFlags.out == "" {
		fmt.Fprintln(cmd.OutOrStdout(), runtimeHex)
	}
	if intr.IsReverted() {
		return fmt.Errorf("constructor execution reverted")
	}
	return nil
}
