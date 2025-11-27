package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/spf13/cobra"
)

var blockFlags struct {
	genesisPath string
	blockPath   string
}

func newBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Block related commands",
	}
	cmd.AddCommand(newBlockApplyCmd())
	cmd.AddCommand(newBlockRunCmd())
	return cmd
}

func newBlockApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply genesis to state (verify genesis loading)",
		RunE:  func(cmd *cobra.Command, args []string) error { return runBlockApply(cmd) },
	}
	cmd.Flags().StringVar(&blockFlags.genesisPath, "genesis", "", "Path to genesis.json")
	return cmd
}

func newBlockRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a block (RLP) on top of genesis",
		RunE:  func(cmd *cobra.Command, args []string) error { return runBlockRun(cmd) },
	}
	cmd.Flags().StringVar(&blockFlags.genesisPath, "genesis", "", "Path to genesis.json")
	cmd.Flags().StringVar(&blockFlags.blockPath, "block", "", "Path to block.rlp")
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
	return nil
}

func runBlockRun(cmd *cobra.Command) error {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	if blockFlags.genesisPath == "" || blockFlags.blockPath == "" {
		return fmt.Errorf("provide --genesis and --block")
	}

	// 1. Load Genesis
	genesis, err := core.LoadGenesis(blockFlags.genesisPath)
	if err != nil {
		return fmt.Errorf("failed to load genesis: %w", err)
	}
	db := core.NewMemoryStateDB()
	if err := genesis.ToStateDB(db); err != nil {
		return fmt.Errorf("failed to apply genesis: %w", err)
	}

	// 2. Load Block
	data, err := os.ReadFile(blockFlags.blockPath)
	if err != nil {
		return fmt.Errorf("failed to read block file: %w", err)
	}
	var block types.Block
	if err := rlp.DecodeBytes(data, &block); err != nil {
		return fmt.Errorf("failed to decode block RLP: %w", err)
	}

	logger.Info().Msgf("Executing block #%d (hash=%s, txs=%d)", block.NumberU64(), block.Hash().Hex(), len(block.Transactions()))

	// 3. Setup Signer
	chainConfig := &params.ChainConfig{
		ChainID:     genesis.Config.ChainID,
		EIP155Block: big.NewInt(0),
	}
	signer := types.MakeSigner(chainConfig, block.Number(), block.Time())

	// 4. Execute Transactions
	for i, tx := range block.Transactions() {
		sender, err := types.Sender(signer, tx)
		if err != nil {
			return fmt.Errorf("tx %d: failed to recover sender: %w", i, err)
		}

		logger.Info().Msgf("Tx %d: %s -> %s (nonce=%d, gas=%d)", i, sender.Hex(), txTo(tx), tx.Nonce(), tx.Gas())

		ret, gasUsed, reverted, err := vm.ApplyTransaction(
			db,
			tx,
			sender,
			block.Number(),
			block.Time(),
			block.Coinbase(),
			block.GasLimit(),
		)
		if err != nil {
			return fmt.Errorf("tx %d failed: %w", i, err)
		}

		status := "success"
		if reverted {
			status = "reverted"
		}
		logger.Info().Msgf("   Result: %s, GasUsed: %d, Return: %x", status, gasUsed, ret)
	}

	logger.Info().Msg("Block execution completed")
	return nil
}

func txTo(tx *types.Transaction) string {
	if tx.To() == nil {
		return "[Contract Creation]"
	}
	return tx.To().Hex()
}
