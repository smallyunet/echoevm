# Gas-bounded arithmetic

Fix `src/SumTo.sol` without changing its public interface.

`sumTo(n)` must return the inclusive sum from 0 through `n` for every
`n <= 1_000_000`, revert for larger values, and use bounded gas independent of
`n`. A call with `n = 1_000_000` must fit comfortably within 20,000 execution
gas.

Keep the patch focused. Compile and execute representative calls, including a
large input, before finishing. Report the validation and gas evidence you
actually collected.
