# Fee quote boundary bug

Fix `src/FeeQuote.sol` without changing its public interface.

`netAfterFee(amount, feeBps)` must subtract a fee expressed in basis points,
where 10,000 bps is 100%. It must accept every fee from 0 through 10,000 bps
and revert for larger values. For example,
`netAfterFee(250000, 250)` must return `243750`.

Keep the patch focused. Compile and execute representative calls before
finishing, then report the validation evidence you actually collected.
