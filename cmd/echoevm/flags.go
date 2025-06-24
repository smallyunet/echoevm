package main

import "flag"

// cliConfig holds command line parameters for echoevm.
type cliConfig struct {
	Bin      string
	Mode     string
	Function string
	Args     string
	Calldata string
	LogLevel string
	RPC      string
	Block    int
}

// parseFlags parses command line flags into a cliConfig.
func parseFlags() *cliConfig {
	cfg := &cliConfig{}
	flag.StringVar(&cfg.Bin, "bin", "", "path to contract .bin file (required)")
	flag.StringVar(&cfg.Mode, "mode", "full", "execution mode: deploy or full")
	flag.StringVar(&cfg.Function, "function", "", "function signature, e.g. 'add(uint256,uint256)'")
	flag.StringVar(&cfg.Args, "args", "", "comma separated arguments for the function")
	flag.StringVar(&cfg.Calldata, "calldata", "", "hex encoded calldata")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "log level: trace, debug, info, warn, error")
	flag.StringVar(&cfg.RPC, "rpc", "https://cloudflare-eth.com", "ethereum RPC endpoint")
	flag.IntVar(&cfg.Block, "block", -1, "block number to execute contract transactions from")
	flag.Parse()
	if cfg.Block == -1 && cfg.Bin == "" {
		flag.Usage()
		panic("-bin flag is required")
	}
	return cfg
}
