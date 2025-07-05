# echoevm

echoevm is a minimal Ethereum Virtual Machine (EVM) implementation in Go, focusing on bytecode execution from the ground up.

## Usage

The `echoevm` command is organised into subcommands to keep flags manageable.
The `run` subcommand executes the constructor contained in a Solidity `.bin`
file and, if runtime code is returned, runs that too. Use the following flags to
customise the behaviour:

```
go run ./cmd/echoevm run -bin path/to/contract.bin -mode [deploy|full] \
        [-calldata HEX | -function "sig" -args "1,2"]
go run ./cmd/echoevm block -block 1 [-rpc URL]
go run ./cmd/echoevm range -start 1 -end 50 [-rpc URL]
```

*Note:* use the directory path (`./cmd/echoevm`) with `go run` so that all
source files are compiled. Running `go run ./cmd/echoevm/main.go` will omit the
flag parsing code located in `flags.go`.

- `run` subcommand:
  - `-bin`  – path to the hex encoded bytecode file (**required**).
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

### Examples

Run with pre-encoded calldata:

```bash
go run ./cmd/echoevm run -mode full -calldata 771602f7...
```

Encode arguments automatically for a function call:

```bash
go run ./cmd/echoevm run -mode full -function 'add(uint256,uint256)' -args "1,2"
```

## Testing

Some example contracts and an integration script are included in the `test/` directory. Run the script below to compile them with `solc` and execute each using `echoevm`:

```bash
./test/run_tests.sh
```
