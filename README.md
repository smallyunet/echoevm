# echoevm

echoevm is a minimal Ethereum Virtual Machine (EVM) implementation in Go, focusing on bytecode execution from the ground up.

## Usage

The `echoevm` command can execute the constructor contained in a Solidity
`.bin` file. By default it runs the deployment bytecode and, if runtime code is
returned, executes that code as well. Use the following flags to customise the
behaviour:

```
./echoevm -bin path/to/contract.bin -mode [deploy|full]
```

- `-bin`  – path to the hex encoded bytecode file (defaults to `build/Add.bin`).
- `-mode` – `deploy` to only run the constructor or `full` to also execute the
  returned runtime code (default `full`).
