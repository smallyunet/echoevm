package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/smallyunet/echoevm/internal/replay"
	"github.com/spf13/cobra"
)

func newWitnessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "witness",
		Short: "Create or inspect standalone EchoEVM replay witnesses",
	}
	cmd.AddCommand(newWitnessImportDebugCmd())
	return cmd
}

func newWitnessImportDebugCmd() *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "import-debug <transaction-hash-or-etherscan-url>",
		Short: "Import prestate from a debug RPC into a standalone witness",
		Long:  "Import prestate through prestateTracer for migration or conformance. The generated witness replays independently; debug RPC is not part of EchoEVM execution.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWitnessImportDebug(cmd, args[0], outputPath)
		},
	}
	cmd.Flags().StringVar(&outputPath, "out", "", "Write witness JSON to this path instead of stdout")
	addVerificationRPCFlag(cmd)
	return cmd
}

func runWitnessImportDebug(cmd *cobra.Command, input, outputPath string) error {
	service, err := replay.NewVerificationService(cmd.Context(), globalFlags.RPCURL)
	if err != nil {
		return err
	}
	witness, err := service.ImportDebugWitness(cmd.Context(), input)
	if err != nil {
		return err
	}
	var out io.Writer = cmd.OutOrStdout()
	var file *os.File
	if outputPath != "" {
		file, err = os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("create replay witness: %w", err)
		}
		defer file.Close()
		out = file
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(witness)
}
