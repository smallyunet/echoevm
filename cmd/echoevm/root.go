package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "echoevm",
		Short: "Bounded causal execution evidence for EVM bytecode",
		Long:  "EchoEVM independently executes Solidity and EVM bytecode and emits bounded causal evidence for people, AI coding agents, CI systems, and editors.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Setup global logger level etc. (can be extended later)
			lvl, err := zerolog.ParseLevel(globalFlags.logLevel)
			if err != nil {
				lvl = zerolog.InfoLevel
			}
			zerolog.SetGlobalLevel(lvl)
			return nil
		},
	}

	globalFlags struct {
		logLevel string
		output   string
		config   string
		RPCURL   string
	}
)

func initRoot() {
	rootCmd.PersistentFlags().StringVarP(&globalFlags.logLevel, "log-level", "L", "info", "Global log level")
	rootCmd.PersistentFlags().StringVarP(&globalFlags.output, "output", "o", "plain", "Output format (plain|json)")
	rootCmd.PersistentFlags().StringVarP(&globalFlags.config, "config", "c", "", "Config file path (optional)")
}

func addSubCommands() {
	rootCmd.AddCommand(newCallCmd())
	rootCmd.AddCommand(newDeployCmd())
	rootCmd.AddCommand(newDisasmCmd())
	rootCmd.AddCommand(newTraceCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newReplCmd())
	rootCmd.AddCommand(newReplayCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newSolidityCmd())
	rootCmd.AddCommand(newWebCmd())
	rootCmd.AddCommand(newWitnessCmd())
}

func execute() {
	initRoot()
	addSubCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
