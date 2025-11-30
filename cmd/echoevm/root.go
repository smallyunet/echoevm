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
		Short: "EchoEVM - lightweight EVM execution & experimentation toolkit",
		Long:  "EchoEVM is a lightweight EVM execution and inspection tool supporting contract deployment, calls, and tracing.",
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
	rootCmd.PersistentFlags().StringVar(&globalFlags.RPCURL, "rpc-url", "https://cloudflare-eth.com", "Default Ethereum RPC endpoint")
}

func addSubCommands() {
	rootCmd.AddCommand(newCallCmd())
	rootCmd.AddCommand(newDeployCmd())
	rootCmd.AddCommand(newTraceCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newReplCmd())
	rootCmd.AddCommand(newRunCmd())
}

func execute() {
	initRoot()
	addSubCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
