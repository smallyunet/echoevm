//! Pure EVM word and signed-arithmetic helpers.

use alloy_primitives::U256;

pub(super) fn is_negative(value: U256) -> bool {
    value.bit(255)
}

pub(super) fn twos_complement(value: U256) -> U256 {
    (!value).wrapping_add(U256::from(1))
}

pub(super) fn signed_div(a: U256, b: U256) -> U256 {
    if b.is_zero() {
        return U256::ZERO;
    }
    let negative = is_negative(a) != is_negative(b);
    let left = if is_negative(a) {
        twos_complement(a)
    } else {
        a
    };
    let right = if is_negative(b) {
        twos_complement(b)
    } else {
        b
    };
    let result = left / right;
    if negative {
        twos_complement(result)
    } else {
        result
    }
}

pub(super) fn signed_mod(a: U256, b: U256) -> U256 {
    if b.is_zero() {
        return U256::ZERO;
    }
    let negative = is_negative(a);
    let left = if negative { twos_complement(a) } else { a };
    let right = if is_negative(b) {
        twos_complement(b)
    } else {
        b
    };
    let result = left % right;
    if negative {
        twos_complement(result)
    } else {
        result
    }
}

pub(super) fn signed_lt(a: U256, b: U256) -> bool {
    match (is_negative(a), is_negative(b)) {
        (true, false) => true,
        (false, true) => false,
        (false, false) => a < b,
        (true, true) => a < b,
    }
}

pub(super) fn arithmetic_shift_right(shift: U256, value: U256) -> U256 {
    let negative = is_negative(value);
    if shift >= U256::from(256) {
        return if negative { U256::MAX } else { U256::ZERO };
    }
    let shift = shift.to::<usize>();
    if shift == 0 || !negative {
        return value >> shift;
    }
    (value >> shift) | (U256::MAX << (256 - shift))
}

pub(super) fn sign_extend(byte: U256, value: U256) -> U256 {
    if byte >= U256::from(32) {
        return value;
    }
    let bit = byte.to::<usize>() * 8 + 7;
    let mask = (U256::from(1) << (bit + 1)) - U256::from(1);
    if value.bit(bit) {
        value | !mask
    } else {
        value & mask
    }
}

pub(super) fn wrapping_pow(mut base: U256, mut exponent: U256) -> U256 {
    let mut result = U256::from(1);
    while !exponent.is_zero() {
        if exponent.bit(0) {
            result = result.wrapping_mul(base);
        }
        exponent >>= 1;
        base = base.wrapping_mul(base);
    }
    result
}
