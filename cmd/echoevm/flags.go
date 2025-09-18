package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/smallyunet/echoevm/internal/config"
	"github.com/smallyunet/echoevm/internal/errors"
)

// cliConfig holds command line parameters for echoevm.
type cliConfig struct {
	Bin        string
	Artifact   string
	Mode       string
	Function   string
	Args       string
	Calldata   string
	LogLevel   string
	RPC        string
	Block      int
	StartBlock int
	EndBlock   int
}

// parseFlags parses subcommand flags and returns the chosen command name along with
// its configuration. Supported subcommands are:
//
//	run   - execute contract bytecode from a .bin file
//	block - execute all contract transactions in a block
//	range - execute a range of blocks
func parseFlags() (string, *cliConfig, error) {
	if len(os.Args) < 2 {
		usage()
		return "", nil, fmt.Errorf("insufficient arguments")
	}

	// Get the subcommand
	subCmd := os.Args[1]

	switch subCmd {
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		cfg := &cliConfig{}
		fs.StringVar(&cfg.Bin, "bin", "", "path to contract .bin file")
		fs.StringVar(&cfg.Artifact, "artifact", "", "path to Hardhat artifact JSON")
		fs.StringVar(&cfg.Mode, "mode", config.DefaultExecutionMode, "execution mode: deploy or full")
		fs.StringVar(&cfg.Function, "function", "", "function signature, e.g. 'add(uint256,uint256)'")
		fs.StringVar(&cfg.Args, "args", "", "comma separated arguments for the function")
		fs.StringVar(&cfg.Calldata, "calldata", "", "hex encoded calldata")
		fs.StringVar(&cfg.LogLevel, "log-level", config.DefaultLogLevel, "log level: trace, debug, info, warn, error")
		fs.Parse(os.Args[2:])
		if cfg.Bin == "" && cfg.Artifact == "" {
			fs.Usage()
			return "", nil, errors.ErrMissingBinOrArtifact
		}
		return "run", cfg, nil

	case "block":
		fs := flag.NewFlagSet("block", flag.ExitOnError)
		cfg := &cliConfig{}
		fs.IntVar(&cfg.Block, "block", -1, "block number to execute contract transactions from")
		fs.StringVar(&cfg.RPC, "rpc", config.DefaultEthereumRPC, "ethereum RPC endpoint")
		fs.StringVar(&cfg.LogLevel, "log-level", config.DefaultLogLevel, "log level: trace, debug, info, warn, error")
		fs.Parse(os.Args[2:])
		if cfg.Block < 0 {
			fs.Usage()
			return "", nil, errors.ErrMissingBlockNumber
		}
		return "block", cfg, nil

	case "range":
		fs := flag.NewFlagSet("range", flag.ExitOnError)
		cfg := &cliConfig{}
		fs.IntVar(&cfg.StartBlock, "start", -1, "start block number for range execution")
		fs.IntVar(&cfg.EndBlock, "end", -1, "end block number for range execution")
		fs.StringVar(&cfg.RPC, "rpc", config.DefaultEthereumRPC, "ethereum RPC endpoint")
		fs.StringVar(&cfg.LogLevel, "log-level", config.DefaultLogLevel, "log level: trace, debug, info, warn, error")
		fs.Parse(os.Args[2:])
		if cfg.StartBlock < 0 || cfg.EndBlock < 0 {
			fs.Usage()
			return "", nil, errors.ErrMissingStartEnd
		}
		if cfg.StartBlock > cfg.EndBlock {
			fs.Usage()
			return "", nil, errors.ErrInvalidRange
		}
		return "range", cfg, nil

	default:
		usage()
		return "", nil, fmt.Errorf("unknown subcommand %s", os.Args[1])
	}
}

// usage prints general help information.
func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <command> [options]\n", os.Args[0])
	fmt.Fprintln(flag.CommandLine.Output(), "Commands:")
	fmt.Fprintln(flag.CommandLine.Output(), "  run    execute contract bytecode from a .bin or Hardhat artifact")
	fmt.Fprintln(flag.CommandLine.Output(), "  block  execute all contract transactions in a block")
	fmt.Fprintln(flag.CommandLine.Output(), "  range  execute a range of blocks")
}
