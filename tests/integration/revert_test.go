package integration

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSnapshotRevert(t *testing.T) {
	statedb, _, _ := setupVM()
	addr := common.HexToAddress("0x5000000000000000000000000000000000000005")
	key := common.Hash{31: 0x01}
	val1 := common.Hash{31: 0xAA}
	val2 := common.Hash{31: 0xBB}

	// 1. Initial State
	statedb.SetState(addr, key, val1)
	statedb.SetNonce(addr, 1)
	statedb.AddBalance(addr, big.NewInt(100))

	// 2. Snapshot
	snap := statedb.Snapshot()

	// 3. Modify State
	statedb.SetState(addr, key, val2)
	statedb.SetNonce(addr, 2)
	statedb.AddBalance(addr, big.NewInt(50)) // Total 150

	// Verify modification happened
	if statedb.GetState(addr, key) != val2 {
		t.Fatal("State update failed")
	}
	if statedb.GetNonce(addr) != 2 {
		t.Fatal("Nonce update failed")
	}
	if statedb.GetBalance(addr).Cmp(big.NewInt(150)) != 0 {
		t.Fatal("Balance update failed")
	}

	// 4. Revert
	statedb.RevertToSnapshot(snap)

	// 5. Verify Revert
	if statedb.GetState(addr, key) != val1 {
		t.Errorf("State revert failed. Got %x, expected %x", statedb.GetState(addr, key), val1)
	}
	if statedb.GetNonce(addr) != 1 {
		t.Errorf("Nonce revert failed. Got %d, expected 1", statedb.GetNonce(addr))
	}
	if statedb.GetBalance(addr).Cmp(big.NewInt(100)) != 0 {
		t.Errorf("Balance revert failed. Got %v, expected 100", statedb.GetBalance(addr))
	}
}
