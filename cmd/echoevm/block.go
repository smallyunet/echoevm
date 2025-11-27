package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/spf13/cobra"
)

var blockFlags struct {
	genesisPath string
}

func newBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Block related commands",
	}
	cmd.AddCommand(newBlockApplyCmd())
	return cmd
}

func newBlockApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply genesis or block to state",
		RunE:  func(cmd *cobra.Command, args []string) error { return runBlockApply(cmd) },
	}
	cmd.Flags().StringVar(&blockFlags.genesisPath, "genesis", "", "Path to genesis.json")
	return cmd
}

func runBlockApply(cmd *cobra.Command) error {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	if blockFlags.genesisPath == "" {
		return fmt.Errorf("provide --genesis")
	}

	logger.Info().Msgf("Loading genesis from %s", blockFlags.genesisPath)
	genesis, err := core.LoadGenesis(blockFlags.genesisPath)
	if err != nil {
		return fmt.Errorf("failed to load genesis: %w", err)
	}

	db := core.NewMemoryStateDB()
	if err := genesis.ToStateDB(db); err != nil {
		return fmt.Errorf("failed to apply genesis to state: %w", err)
	}

	logger.Info().Msgf("Genesis applied successfully. ChainID: %v", genesis.Config.ChainID)
	// We can't easily count accounts in MemoryStateDB without exposing map or adding a method.
	// But we can just say success.

	return nil
}
