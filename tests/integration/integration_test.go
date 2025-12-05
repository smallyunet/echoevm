package integration

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

// Helper to setup a fresh VM environment
func setupVM() (*core.MemoryStateDB, common.Address, common.Address) {
	statedb := core.NewMemoryStateDB()
	sender := common.HexToAddress("0x1000000000000000000000000000000000000001")
	receiver := common.HexToAddress("0x2000000000000000000000000000000000000002")

	// Fund sender
	statedb.AddBalance(sender, big.NewInt(1000000000000000000)) // 1 ETH

	return statedb, sender, receiver
}

func TestValueTransfer(t *testing.T) {
	statedb, sender, receiver := setupVM()

	// Transfer 1000 wei from sender to receiver
	amount := big.NewInt(1000)

	// In a real transaction we would have gas, etc.
	// Here we are testing the VM / StateDB interaction directly or via a minimal run.
	// Since the Interpreter runs *code*, a simple value transfer is usually handled
	// by the transaction processor *before* entering the VM (for EOA to EOA).
	// However, we can test the StateDB directly here as part of integration.

	if statedb.GetBalance(sender).Cmp(big.NewInt(1000000000000000000)) != 0 {
		t.Fatalf("Initial balance mismatch")
	}

	statedb.SubBalance(sender, amount)
	statedb.AddBalance(receiver, amount)

	if statedb.GetBalance(sender).Cmp(big.NewInt(999999999999999000)) != 0 {
		t.Errorf("Sender balance incorrect after transfer")
	}
	if statedb.GetBalance(receiver).Cmp(amount) != 0 {
		t.Errorf("Receiver balance incorrect after transfer")
	}
}

func TestContractDeploymentAndCall(t *testing.T) {
	statedb, sender, _ := setupVM()

	// 1. Deploy a simple contract that returns 42
	// PUSH1 42 PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN
	// Runtime code: 602a60005260206000f3
	runtimeCode := []byte{0x60, 0x2a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}

	// Deployment code (returns the runtime code)
	// PUSH1 10 (length) PUSH1 0 (offset) RETURN
	// Wait, we need to copy code to memory first?
	// Simplified: We can just set the code directly in StateDB for this test
	// to verify the VM execution of the runtime code.

	contractAddr := common.HexToAddress("0x3000000000000000000000000000000000000003")
	statedb.SetCode(contractAddr, runtimeCode)

	// 2. Call the contract
	interpreter := vm.New(runtimeCode, statedb, contractAddr)
	interpreter.SetCaller(sender)
	interpreter.SetBlockGasLimit(100000)
	interpreter.SetGas(100000)

	interpreter.Run()

	if interpreter.Err() != nil {
		t.Fatalf("Execution failed: %v", interpreter.Err())
	}

	ret := interpreter.ReturnedCode() // In this simple case, it returns 32 bytes of memory
	if len(ret) < 32 {
		t.Fatalf("Return data too short")
	}

	result := new(big.Int).SetBytes(ret)
	if result.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("Expected 42, got %v", result)
	}
}

func TestSstoreSload(t *testing.T) {
	statedb, sender, _ := setupVM()
	contractAddr := common.HexToAddress("0x4000000000000000000000000000000000000004")

	// Contract: SSTORE(key=1, val=123) then SLOAD(key=1)
	// PUSH1 123 PUSH1 1 SSTORE PUSH1 1 SLOAD PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN
	// 607b 6001 55 6001 54 6000 52 6020 6000 f3
	code := []byte{
		0x60, 0x7b, // PUSH1 123
		0x60, 0x01, // PUSH1 1
		0x55,       // SSTORE
		0x60, 0x01, // PUSH1 1
		0x54,       // SLOAD
		0x60, 0x00, // PUSH1 0
		0x52,       // MSTORE
		0x60, 0x20, // PUSH1 32
		0x60, 0x00, // PUSH1 0
		0xf3, // RETURN
	}

	statedb.SetCode(contractAddr, code)

	interpreter := vm.New(code, statedb, contractAddr)
	interpreter.SetCaller(sender)
	interpreter.SetBlockGasLimit(100000)
	interpreter.SetGas(100000)

	interpreter.Run()

	if interpreter.Err() != nil {
		t.Fatalf("Execution failed: %v", interpreter.Err())
	}

	// Verify Storage
	key := common.Hash{31: 1} // 0x...01
	val := statedb.GetState(contractAddr, key)
	expectedVal := common.BigToHash(big.NewInt(123))

	if val != expectedVal {
		t.Errorf("Storage mismatch. Expected %x, got %x", expectedVal, val)
	}

	// Verify Return
	ret := interpreter.ReturnedCode()
	res := new(big.Int).SetBytes(ret)
	if res.Cmp(big.NewInt(123)) != 0 {
		t.Errorf("Return mismatch. Expected 123, got %v", res)
	}
}
