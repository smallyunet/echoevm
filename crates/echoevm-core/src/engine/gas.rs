//! Shared interpreter gas helpers.

pub(super) const fn words(size: usize) -> u64 {
    size.div_ceil(32) as u64
}

pub(super) const fn memory_cost(size: usize) -> u64 {
    let words = words(size);
    3u64.saturating_mul(words)
        .saturating_add(words.saturating_mul(words) / 512)
}

pub(super) const fn copy_gas(size: usize) -> u64 {
    3u64.saturating_mul(words(size))
}

pub(super) const fn environment_gas(op: u8) -> u64 {
    match op {
        0x44 | 0x47 => 5,
        0x40 => 20,
        0x49 => 3,
        _ => 2,
    }
}
