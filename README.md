# echoevm

echoevm is a minimal Ethereum Virtual Machine (EVM) implementation in Go, focusing on bytecode execution from the ground up.

## Usage

The `echoevm` command can execute the constructor contained in a Solidity
`.bin` file. By default it runs the deployment bytecode and, if runtime code is
returned, executes that code as well. Use the following flags to customise the
behaviour:

```
./echoevm -bin path/to/contract.bin -mode [deploy|full] \ 
          [-calldata HEX | -function "sig" -args "1,2"]
```

- `-bin`  – path to the hex encoded bytecode file (defaults to `build/Add.bin`).
- `-mode` – `deploy` to only run the constructor or `full` to also execute the
  returned runtime code (default `full`).
- `-calldata` – hex-encoded calldata to supply when running the runtime code.
- `-function`/`-args` – alternatively specify a function signature and comma
  separated arguments (e.g. `-function "add(uint256,uint256)" -args "1,2"`)
  which will be ABI encoded automatically.

### Examples

Run with pre-encoded calldata:

```bash
go run ./cmd/echoevm -mode full -calldata 771602f7...
```

Encode arguments automatically for a function call:

```bash
go run ./cmd/echoevm -mode full -function 'add(uint256,uint256)' -args "1,2"
```
