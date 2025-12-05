package core

import (
	"github.com/ethereum/go-ethereum/common"
)

type accessListAddChange struct {
	addr common.Address
}

func (ch accessListAddChange) revert(db *MemoryStateDB) {
	delete(db.accessListAddrs, ch.addr)
}

type accessListSlotChange struct {
	addr common.Address
	slot common.Hash
}

func (ch accessListSlotChange) revert(db *MemoryStateDB) {
	if slots, ok := db.accessListSlots[ch.addr]; ok {
		delete(slots, ch.slot)
	}
}

func (db *MemoryStateDB) AddAddressToAccessList(addr common.Address) {
	if _, ok := db.accessListAddrs[addr]; !ok {
		db.journal = append(db.journal, accessListAddChange{addr: addr})
		db.accessListAddrs[addr] = struct{}{}
	}
}

func (db *MemoryStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	if _, ok := db.accessListSlots[addr]; !ok {
		db.accessListSlots[addr] = make(map[common.Hash]struct{})
	}
	if _, ok := db.accessListSlots[addr][slot]; !ok {
		db.journal = append(db.journal, accessListSlotChange{addr: addr, slot: slot})
		db.accessListSlots[addr][slot] = struct{}{}
	}
}

func (db *MemoryStateDB) AddressInAccessList(addr common.Address) bool {
	_, ok := db.accessListAddrs[addr]
	return ok
}

func (db *MemoryStateDB) SlotInAccessList(addr common.Address, slot common.Hash) bool {
	slots, ok := db.accessListSlots[addr]
	if !ok {
		return false
	}
	_, okSlot := slots[slot]
	return okSlot
}
