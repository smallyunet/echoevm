# Packed payload decoder bug

Fix `src/PackedDecoder.sol` without changing its public interface.

The 32-byte wire format is exactly `abi.encodePacked(uint96 amount, address
recipient)`: the first 12 bytes are the big-endian amount and the last 20 bytes
are the recipient. `decode` must return those two values and reject every input
whose length is not exactly 32 bytes.

Keep the patch focused. Compile and execute representative calls before
finishing, then report the validation evidence you actually collected.
