package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/smallyunet/echoevm/internal/differential"
	webui "github.com/smallyunet/echoevm/internal/web"
	"github.com/spf13/cobra"
)

type diffFlags struct {
	code, input, fork, format, addr string
	gas                             uint64
	web                             bool
}

func newDiffCmd() *cobra.Command {
	flags := &diffFlags{}
	cmd := &cobra.Command{
		Use:     "diff",
		Short:   "Compare EchoEVM with embedded Geth under Cancun rules",
		Example: "echoevm diff --code 60026003015f5260205ff3 --input 0x --gas 1000000\nechoevm diff --web --addr :8080",
		RunE:    func(cmd *cobra.Command, _ []string) error { return runDiff(cmd.Context(), cmd.OutOrStdout(), flags) },
	}
	cmd.Flags().StringVar(&flags.code, "code", "", "EVM bytecode as hex")
	cmd.Flags().StringVar(&flags.input, "input", "0x", "calldata as hex")
	cmd.Flags().Uint64Var(&flags.gas, "gas", differential.DefaultGasLimit, "execution gas limit")
	cmd.Flags().StringVar(&flags.fork, "fork", differential.ForkCancun, "EVM fork (Cancun only)")
	cmd.Flags().StringVar(&flags.format, "format", "text", "output format (text|json)")
	cmd.Flags().BoolVar(&flags.web, "web", false, "start the local Differential Explorer")
	cmd.Flags().StringVar(&flags.addr, "addr", ":8080", "HTTP listen address for --web")
	return cmd
}

func runDiff(ctx context.Context, out io.Writer, flags *diffFlags) error {
	engine := differential.DefaultEngine()
	if flags.web {
		return webui.NewDifferentialServer(flags.addr, engine).Start()
	}
	if strings.TrimSpace(flags.code) == "" {
		return fmt.Errorf("--code is required unless --web is used")
	}
	if flags.format != "text" && flags.format != "json" {
		return fmt.Errorf("unsupported format %q: use text or json", flags.format)
	}
	result, err := engine.Compare(ctx, differential.Request{
		Fork: flags.fork, Bytecode: flags.code, Calldata: flags.input, GasLimit: flags.gas,
	})
	if err != nil {
		return err
	}
	if flags.format == "json" {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	writeDiffText(out, result)
	return nil
}

func writeDiffText(out io.Writer, result differential.ComparisonResult) {
	verdict := "MATCH"
	if !result.Match {
		verdict = "DIVERGENCE"
	}
	fmt.Fprintf(out, "%s — EchoEVM vs Geth (%s, isolated memory state)\n", verdict, result.Request.Fork)
	fmt.Fprintf(out, "status  echo=%s geth=%s match=%t\n", result.EchoEVM.Status, result.Geth.Status, result.StatusMatch)
	fmt.Fprintf(out, "return  echo=%s geth=%s match=%t\n", result.EchoEVM.ReturnData, result.Geth.ReturnData, result.ReturnDataMatch)
	fmt.Fprintf(out, "gas     echo=%d geth=%d match=%t\n", result.EchoEVM.GasUsed, result.Geth.GasUsed, result.GasMatch)
	fmt.Fprintf(out, "storage match=%t  trace match=%t steps=%d/%d\n", result.StorageMatch, result.TraceMatch, len(result.EchoEVM.Trace), len(result.Geth.Trace))
	if d := result.FirstDivergence; d != nil {
		location := "result"
		if d.Step != nil {
			location = fmt.Sprintf("step=%d", *d.Step)
			if d.PC != nil {
				location += fmt.Sprintf(" pc=%d", *d.PC)
			}
			if d.Opcode != "" {
				location += " opcode=" + d.Opcode
			}
		}
		left, _ := json.Marshal(d.EchoEVM)
		right, _ := json.Marshal(d.Geth)
		fmt.Fprintf(out, "first divergence: %s field=%s\n  EchoEVM: %s\n  Geth:    %s\n", location, d.Field, left, right)
	}
	fmt.Fprintf(out, "Scope: this input matched only in the environment shown; this is not a claim of complete EVM compatibility.\n")
}
