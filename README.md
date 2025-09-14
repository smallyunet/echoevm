# echoevm

echoevm is a minimal Ethereum Virtual Machine (EVM) implementation in Go, focusing on bytecode execution from the ground up. It supports executing smart contracts, handling various opcodes, and provides comprehensive testing capabilities.

## Features

- **Bytecode Execution**: Execute Solidity bytecode directly from `.bin` files or Hardhat artifacts
- **Function Calls**: Automatic ABI encoding for function calls with type support
- **Error Handling**: Proper handling of REVERT conditions with appropriate exit codes
- **Testing Framework**: Comprehensive test suites for validation and debugging
- **Block Processing**: Fetch and execute blocks from Ethereum networks via RPC
- **Logging**: Detailed execution tracing with configurable log levels
- **JSON-RPC API**: Geth-compatible RPC server for seamless integration with existing Ethereum tools

## Requirements

- Go 1.23.2 or later
- (Optional) Solidity compiler for compiling test contracts

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/smallyunet/echoevm.git
cd echoevm

# Build the binary
make build

# Or install to GOPATH/bin
make install
```

## Usage

The `echoevm` command is organised into subcommands to keep flags manageable.
The `run` subcommand executes the constructor contained in a Solidity `.bin`
file and, if runtime code is returned, runs that too. Use the following flags to
customise the behaviour:

```
go run ./cmd/echoevm run -bin path/to/contract.bin -mode [deploy|full] \
       [-calldata HEX | -function "sig" -args "1,2"]
go run ./cmd/echoevm run -artifact path/to/Contract.json -mode [deploy|full] \
       [-calldata HEX | -function "sig" -args "1,2"]
go run ./cmd/echoevm block -block 1 [-rpc URL]
go run ./cmd/echoevm range -start 1 -end 50 [-rpc URL]
go run ./cmd/echoevm serve [-http localhost:8545]
```

*Note:* use the directory path (`./cmd/echoevm`) with `go run` so that all
source files are compiled. Running `go run ./cmd/echoevm/main.go` will omit the
flag parsing code located in `flags.go`.

- `run` subcommand:
  - `-bin`  – path to the hex encoded bytecode file.
  - `-artifact` – path to a Hardhat artifact JSON file containing bytecode.
  - `-mode` – `deploy` to only run the constructor or `full` to also execute the
    returned runtime code (default `full`).
  - `-calldata` – hex-encoded calldata for the runtime code.
  - `-function`/`-args` – alternatively specify a function signature and comma
    separated arguments (e.g. `-function "add(uint256,uint256)" -args "1,2"`)
    which will be ABI encoded automatically.
- `block` subcommand:
  - `-block`/`-rpc` – fetch a single block via RPC. By default `-rpc` uses
    `https://cloudflare-eth.com`.
- `range` subcommand:
  - `-start`/`-end`/`-rpc` – execute a range of blocks via RPC.
- `serve` subcommand:
  - `-http` – HTTP RPC endpoint address (default: localhost:8545). Starts a JSON-RPC server
    compatible with Ethereum clients.

### Examples

#### Basic Function Calls

Run a simple addition function:

```bash
go run ./cmd/echoevm run \
  -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json \
  -function "add(uint256,uint256)" -args "42,58"
```

Check a boolean state:

```bash
go run ./cmd/echoevm run \
  -artifact ./test/contract/artifacts/contracts/01-data-types/BoolType.sol/BoolType.json \
  -function "isActive()"
```

#### Using Pre-encoded Calldata

```bash
go run ./cmd/echoevm run -mode full -calldata 771602f7000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000002
```

#### Testing Error Conditions

Test a require statement (will exit with code 1):

```bash
go run ./cmd/echoevm run \
  -artifact ./test/contract/artifacts/contracts/03-control-flow/Require.sol/Require.json \
  -function "test(uint256)" -args "0"
```

#### Complex Contract Execution

Run a factorial calculation:

```bash
go run ./cmd/echoevm run \
  -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json \
  -function "fact(uint256)" -args "5"
```

Execute control flow with loops:

```bash
go run ./cmd/echoevm run \
  -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json \
  -function "forLoop(uint256)" -args "10"
```

#### Block Processing

Fetch and process a specific block:

```bash
go run ./cmd/echoevm block -block 18000000 -rpc https://mainnet.infura.io/v3/YOUR-PROJECT-ID
```

Process a range of blocks:

```bash
go run ./cmd/echoevm range -start 1 -end 10 -rpc https://cloudflare-eth.com
```

#### JSON-RPC Server

Start a Geth-compatible JSON-RPC server:

```bash
go run ./cmd/echoevm serve -http localhost:8545
```

You can then use any Ethereum client tools (like web3.js, ethers.js, or web3.py) to connect to your local echoevm instance as if it were a real Ethereum node.

## Testing

The project includes comprehensive test suites to validate EVM functionality:

### Quick Start

```bash
# Run all tests
./test/test.sh

# Run only binary tests (fast)
./test/test.sh --binary

# Run only contract tests (comprehensive)
./test/test.sh --contract

# Run with verbose output
./test/test.sh --verbose

# Show help
./test/test.sh --help
```

### Test Structure

- **Single Script**: One `test/test.sh` script handles all testing
- **Binary Tests** (`test/binary/`): Quick tests using pre-compiled bytecode
- **Contract Tests** (`test/contract/`): Full contract execution tests with Hardhat artifacts

### Test Coverage

The test suite covers:

- **Data Types**: Integer operations, boolean logic, string handling
- **Arithmetic**: Addition, subtraction, multiplication, division
- **Control Flow**: If-else conditions, loops, require statements
- **Edge Cases**: Large numbers, zero values, division by zero
- **Error Handling**: REVERT conditions, invalid inputs
- **Performance**: Loop execution, factorial calculations

For detailed testing documentation, see [test/README.md](test/README.md).

## Architecture

### Core Components

- **EVM Core** (`internal/evm/core/`): Stack, memory, and opcode implementations
- **Interpreter** (`internal/evm/vm/`): Virtual machine execution engine
- **CLI** (`cmd/echoevm/`): Command-line interface and argument parsing

### Supported Opcodes

The implementation supports essential EVM opcodes including:

- **Arithmetic**: ADD, SUB, MUL, DIV, MOD
- **Bitwise**: AND, OR, XOR, NOT, SHL, SHR
- **Comparison**: LT, GT, EQ, ISZERO
- **Stack**: POP, PUSH, DUP, SWAP
- **Memory**: MLOAD, MSTORE, MSIZE
- **Storage**: SLOAD, SSTORE
- **Control**: JUMP, JUMPI, JUMPDEST
- **Environment**: ADDRESS, BALANCE, ORIGIN, CALLER
- **Execution**: CALL, RETURN, REVERT, STOP

## Exit Codes

- **0**: Successful execution
- **1**: Contract execution reverted (REVERT opcode encountered)
- **2**: Invalid arguments or configuration error

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass with `make test-all`
5. Submit a pull request

## License

This project is open source. Please check the license file for details.
