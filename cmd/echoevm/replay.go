package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/smallyunet/echoevm/internal/replay"
	"github.com/spf13/cobra"
)

func newReplayCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "replay <transaction-hash-or-etherscan-url>",
		Short: "Replay a confirmed Ethereum transaction with RPC prestate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" {
				return fmt.Errorf("unsupported format %q: use text or json", format)
			}
			return runReplay(cmd.Context(), cmd.OutOrStdout(), args[0], format)
		},
		Example: "echoevm replay 0xabc... --rpc-url https://your-trace-rpc.example --format json",
	}
	cmd.Flags().StringVar(&format, "format", "text", "output format (text|json)")
	return cmd
}

func runReplay(ctx context.Context, out io.Writer, input, format string) error {
	service, err := replay.NewService(ctx, globalFlags.RPCURL)
	if err != nil {
		return err
	}
	result, err := service.Replay(ctx, replay.Request{Input: input})
	if err != nil {
		return err
	}
	if format == "json" {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	verdict := "MATCH"
	if !result.Match {
		verdict = "DIVERGENCE"
	}
	if _, err := fmt.Fprintf(out, "%s — EchoEVM transaction replay\n", verdict); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "tx      %s block=%d fork=%s\n", result.Transaction.Hash, result.Transaction.BlockNumber, result.Transaction.Fork); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "status  echo=%s geth=%s match=%t\n", result.EchoEVM.Status, result.Geth.Status, result.StatusMatch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "return  echo=%s geth=%s match=%t\n", result.EchoEVM.ReturnData, result.Geth.ReturnData, result.ReturnDataMatch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "gas     echo=%d geth=%d match=%t\n", result.EchoEVM.GasUsed, result.Geth.GasUsed, result.GasMatch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "state   match=%t fields=%d\n", result.StateMatch, len(result.EchoState)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "trace   match=%t steps=%d/%d\n", result.TraceMatch, len(result.EchoEVM.Trace), len(result.Geth.Trace)); err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		if _, err := fmt.Fprintln(out, "warning "+warning); err != nil {
			return err
		}
	}
	return nil
}
