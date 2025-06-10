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
}

// parseFlags parses command line flags into a cliConfig.
func parseFlags() *cliConfig {
	cfg := &cliConfig{}
	flag.StringVar(&cfg.Bin, "bin", "build/Add.bin", "path to contract .bin file")
	flag.StringVar(&cfg.Mode, "mode", "full", "execution mode: deploy or full")
	flag.StringVar(&cfg.Function, "function", "", "function signature, e.g. 'add(uint256,uint256)'")
	flag.StringVar(&cfg.Args, "args", "", "comma separated arguments for the function")
	flag.StringVar(&cfg.Calldata, "calldata", "", "hex encoded calldata")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "log level: trace, debug, info, warn, error")
	flag.Parse()
	return cfg
}
