# CREATE2 prediction mismatch

Fix `src/Create2Factory.sol` without changing its public interfaces.

`predicted(salt, seed)` must return the exact address at which `deploy` creates
`Child` for the same arguments. `deployAndCheck` must therefore return true for
arbitrary salts and seeds. Preserve the child's constructor value.

Keep the patch focused. Compile and execute `deployAndCheck` with representative
arguments before finishing, then report the validation evidence you actually
collected.
