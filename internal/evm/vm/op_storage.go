package vm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

func opSload(i *Interpreter, _ byte) {
	keyBig := i.stack.PopSafe()
	key := common.BigToHash(keyBig)

	// GasTable[SLOAD] = 800 (GasSload)
	// EIP-2929: Warm = 100, Cold = 2100

	if i.statedb.SlotInAccessList(i.address, key) {
		// Warm: 100. Already paid 800. Refund 700.
		i.gas += 700
	} else {
		// Cold: 2100. Already paid 800. Pay 1300.
		extra := uint64(1300)
		if i.gas < extra {
			i.err = fmt.Errorf("out of gas: sload")
			i.reverted = true
			return
		}
		i.gas -= extra
		i.statedb.AddSlotToAccessList(i.address, key)
	}

	val := i.statedb.GetState(i.address, key)
	i.stack.PushSafe(val.Big())
}

func opSstore(i *Interpreter, _ byte) {
	keyBig := i.stack.PopSafe()
	valBig := i.stack.PopSafe()
	key := common.BigToHash(keyBig)
	value := common.BigToHash(valBig)

	// GasTable[SSTORE] = 0.

	var cost uint64
	// 1. Cold SLOAD cost
	if !i.statedb.SlotInAccessList(i.address, key) {
		cost += 2100
		i.statedb.AddSlotToAccessList(i.address, key)
	} else {
		cost += 100
	}

	// 2. Dynamic cost (EIP-2200)
	current := i.statedb.GetState(i.address, key)
	original := i.statedb.GetOriginalState(i.address, key)

	if current == value {
		// No-op (value unchanged)
		cost += 100 // SLOAD_GAS (warm)
	} else {
		if original == current {
			// Slot is clean (original == current)
			if original == (common.Hash{}) {
				// Init: 0 -> Non-zero
				cost += 20000
			} else {
				// Clean update: Non-zero -> ...
				cost += 2900
			}
		} else {
			// Slot is dirty (original != current)
			cost += 100 // SLOAD_GAS (warm)
		}
	}

	// Refunds
	if current != value {
		if original == current {
			// Clean -> Dirty
			if original != (common.Hash{}) && value == (common.Hash{}) {
				// Clearing: Non-zero -> 0
				i.statedb.AddRefund(4800)
			}
		} else {
			// Dirty -> Dirty
			if original != (common.Hash{}) {
				if current == (common.Hash{}) {
					// Was cleared, now set back: 0 -> Non-zero
					i.statedb.SubRefund(4800)
				} else if value == (common.Hash{}) {
					// Was set, now cleared: Non-zero -> 0
					i.statedb.AddRefund(4800)
				}
			}
			if original == value {
				// Reset to original
				if original == (common.Hash{}) {
					// Original was 0. We paid 20000 for init.
					// Now we are back to 0.
					// Refund: 20000 - 100 (warm sload) = 19900
					i.statedb.AddRefund(19900)
				} else {
					// Original was non-zero. We paid 2900 for update.
					// Now we are back to original.
					// Refund: 2900 - 100 (warm sload) = 2800
					i.statedb.AddRefund(2800)
				}
			}
		}
	}

	if i.gas < cost {
		i.err = fmt.Errorf("out of gas: sstore")
		i.reverted = true
		return
	}
	i.gas -= cost

	i.statedb.SetState(i.address, key, value)
}
