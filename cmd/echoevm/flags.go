package main

import "flag"

// cliConfig holds command line parameters for echoevm.
type cliConfig struct {
	Bin        string
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
	flag.IntVar(&cfg.StartBlock, "start-block", -1, "start block number for range execution")
	flag.IntVar(&cfg.EndBlock, "end-block", -1, "end block number for range execution")
	flag.Parse()
	if (cfg.StartBlock >= 0 || cfg.EndBlock >= 0) && !(cfg.StartBlock >= 0 && cfg.EndBlock >= 0) {
		flag.Usage()
		panic("both -start-block and -end-block must be provided")
	}
	if cfg.StartBlock >= 0 && cfg.EndBlock >= 0 && cfg.StartBlock > cfg.EndBlock {
		flag.Usage()
		panic("-start-block must be less than or equal to -end-block")
	}
	if cfg.Block == -1 && cfg.StartBlock == -1 && cfg.EndBlock == -1 && cfg.Bin == "" {
		flag.Usage()
		panic("-bin flag is required")
	}
	return cfg
}
