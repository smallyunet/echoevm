package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/smallyunet/echoevm/internal/replay"
	"github.com/spf13/cobra"
)

func newReplayCmd() *cobra.Command {
	var format string
	var profile string
	var limit int
	var maxMemoryBytes int
	cmd := &cobra.Command{
		Use:   "replay <witness.json>",
		Short: "Replay a self-contained transaction witness with EchoEVM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" && format != "evidence-json" {
				return fmt.Errorf("unsupported format %q: use text, json, or evidence-json", format)
			}
			return runReplay(cmd.Context(), cmd.OutOrStdout(), args[0], format, profile, limit, maxMemoryBytes)
		},
		Example: "echoevm replay ./transaction.witness.json --format evidence-json --profile auto --limit 40",
	}
	cmd.Flags().StringVar(&format, "format", "text", "output format (text|json|evidence-json)")
	cmd.Flags().StringVar(&profile, "profile", "auto", "Evidence profile for evidence-json (auto|revert|storage|call|abi|gas|arithmetic|full)")
	cmd.Flags().IntVar(&limit, "limit", replay.DefaultEvidenceLimit, "Maximum evidence events; replay still completes (0 = no limit)")
	cmd.Flags().IntVar(&maxMemoryBytes, "max-memory-bytes", replay.DefaultEvidenceMemoryBytes, "Maximum changed memory bytes captured per opcode")
	return cmd
}

func runReplay(ctx context.Context, out io.Writer, witnessPath, format, profile string, limit, maxMemoryBytes int) error {
	witness, err := replay.LoadWitness(witnessPath)
	if err != nil {
		return err
	}
	req := replay.ReplayRequest{Witness: witness}
	if format == "evidence-json" {
		req.Profile, req.Limit, req.MaxMemoryBytes = profile, limit, maxMemoryBytes
	}
	result, err := replay.ReplayWitness(ctx, req)
	if err != nil {
		return err
	}
	if format == "json" || format == "evidence-json" {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if format == "evidence-json" {
			if result.Evidence == nil {
				return errors.New("missing replay evidence document")
			}
			return encoder.Encode(result.Evidence)
		}
		return encoder.Encode(result)
	}
	if _, err := fmt.Fprintln(out, "EXECUTED — EchoEVM standalone transaction replay"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "tx       %s block=%d fork=%s\n", result.Transaction.Hash, result.Transaction.BlockNumber, result.Transaction.Fork); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "status   %s\n", result.Execution.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "return   %s\n", result.Execution.ReturnData); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "gas      %d\n", result.Execution.GasUsed); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "trace    steps=%d\n", len(result.Execution.Trace)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "witness  %s sha256=%s\n", result.Witness.Schema, result.Witness.SHA256); err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		if _, err := fmt.Fprintln(out, "warning  "+warning); err != nil {
			return err
		}
	}
	return nil
}
