# echoevm

echoevm is a minimal Ethereum Virtual Machine (EVM) implementation in Go, focusing on bytecode execution from the ground up.

## Usage

The `echoevm` command can execute the constructor contained in a Solidity
`.bin` file. By default it runs the deployment bytecode and, if runtime code is
returned, executes that code as well. Use the following flags to customise the
behaviour:

```
go run ./cmd/echoevm -bin path/to/contract.bin -mode [deploy|full] \
        [-calldata HEX | -function "sig" -args "1,2"]
go run ./cmd/echoevm -block 1 [-rpc URL]
go run ./cmd/echoevm -start-block 1 -end-block 50 [-rpc URL]
```

*Note:* use the directory path (`./cmd/echoevm`) with `go run` so that all
source files are compiled. Running `go run ./cmd/echoevm/main.go` will omit the
flag parsing code located in `flags.go`.

- `-bin`  – path to the hex encoded bytecode file (**required**).
- `-mode` – `deploy` to only run the constructor or `full` to also execute the
  returned runtime code (default `full`).
- `-calldata` – hex-encoded calldata to supply when running the runtime code.
- `-function`/`-args` – alternatively specify a function signature and comma
  separated arguments (e.g. `-function "add(uint256,uint256)" -args "1,2"`)
  which will be ABI encoded automatically. One of `-calldata` or `-function`/`-args` is required when running in `full` mode.
- `-block`/`-rpc` – fetch a block via RPC and execute all contract transactions
  it contains. By default `-rpc` uses `https://cloudflare-eth.com`.
  The CLI prints the block number, how many contract transactions were found and
  how many executed successfully.
- `-start-block`/`-end-block` – execute a range of blocks via RPC.

### Examples

Run with pre-encoded calldata:

```bash
go run ./cmd/echoevm -mode full -calldata 771602f7...
```

Encode arguments automatically for a function call:

```bash
go run ./cmd/echoevm -mode full -function 'add(uint256,uint256)' -args "1,2"
```

## Testing

Some example contracts and an integration script are included in the `test/` directory. Run the script below to compile them with `solc` and execute each using `echoevm`:

```bash
./test/run_tests.sh
```
