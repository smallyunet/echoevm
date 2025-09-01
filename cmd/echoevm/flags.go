package main

import (
	"flag"
	"fmt"
	"os"
)

// cliConfig holds command line parameters for echoevm.
type cliConfig struct {
	Bin         string
	Artifact    string
	Mode        string
	Function    string
	Args        string
	Calldata    string
	LogLevel    string
	RPC         string
	RPCEndpoint string
	Block       int
	StartBlock  int
	EndBlock    int
}

// parseFlags parses subcommand flags and returns the chosen command name along with
// its configuration. Supported subcommands are:
//
//	run   - execute contract bytecode from a .bin file
//	block - execute all contract transactions in a block
//	range - execute a range of blocks
func parseFlags() (string, *cliConfig) {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	// Get the subcommand
	subCmd := os.Args[1]

	switch subCmd {
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		cfg := &cliConfig{}
		fs.StringVar(&cfg.Bin, "bin", "", "path to contract .bin file")
		fs.StringVar(&cfg.Artifact, "artifact", "", "path to Hardhat artifact JSON")
		fs.StringVar(&cfg.Mode, "mode", "full", "execution mode: deploy or full")
		fs.StringVar(&cfg.Function, "function", "", "function signature, e.g. 'add(uint256,uint256)'")
		fs.StringVar(&cfg.Args, "args", "", "comma separated arguments for the function")
		fs.StringVar(&cfg.Calldata, "calldata", "", "hex encoded calldata")
		fs.StringVar(&cfg.LogLevel, "log-level", "info", "log level: trace, debug, info, warn, error")
		fs.Parse(os.Args[2:])
		if cfg.Bin == "" && cfg.Artifact == "" {
			fs.Usage()
			panic("-bin or -artifact flag is required")
		}
		return "run", cfg

	case "block":
		fs := flag.NewFlagSet("block", flag.ExitOnError)
		cfg := &cliConfig{}
		fs.IntVar(&cfg.Block, "block", -1, "block number to execute contract transactions from")
		fs.StringVar(&cfg.RPC, "rpc", "https://cloudflare-eth.com", "ethereum RPC endpoint")
		fs.StringVar(&cfg.LogLevel, "log-level", "info", "log level: trace, debug, info, warn, error")
		fs.Parse(os.Args[2:])
		if cfg.Block < 0 {
			fs.Usage()
			panic("-block must be provided")
		}
		return "block", cfg

	case "range":
		fs := flag.NewFlagSet("range", flag.ExitOnError)
		cfg := &cliConfig{}
		fs.IntVar(&cfg.StartBlock, "start", -1, "start block number for range execution")
		fs.IntVar(&cfg.EndBlock, "end", -1, "end block number for range execution")
		fs.StringVar(&cfg.RPC, "rpc", "https://cloudflare-eth.com", "ethereum RPC endpoint")
		fs.StringVar(&cfg.LogLevel, "log-level", "info", "log level: trace, debug, info, warn, error")
		fs.Parse(os.Args[2:])
		if cfg.StartBlock < 0 || cfg.EndBlock < 0 {
			fs.Usage()
			panic("both -start and -end must be provided")
		}
		if cfg.StartBlock > cfg.EndBlock {
			fs.Usage()
			panic("-start must be less than or equal to -end")
		}
		return "range", cfg

	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		cfg := &cliConfig{}
		fs.StringVar(&cfg.RPCEndpoint, "http", "localhost:8545", "HTTP RPC endpoint address")
		fs.StringVar(&cfg.LogLevel, "log-level", "info", "log level: trace, debug, info, warn, error")
		fs.Parse(os.Args[2:])
		return "serve", cfg

	default:
		usage()
		fmt.Fprintf(os.Stderr, "unknown subcommand %s\n", os.Args[1])
		os.Exit(1)
	}

	return "", nil // unreachable
}

// usage prints general help information.
func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <command> [options]\n", os.Args[0])
	fmt.Fprintln(flag.CommandLine.Output(), "Commands:")
	fmt.Fprintln(flag.CommandLine.Output(), "  run    execute contract bytecode from a .bin or Hardhat artifact")
	fmt.Fprintln(flag.CommandLine.Output(), "  block  execute all contract transactions in a block")
	fmt.Fprintln(flag.CommandLine.Output(), "  range  execute a range of blocks")
	fmt.Fprintln(flag.CommandLine.Output(), "  serve  start a JSON-RPC server compatible with Geth")
}
