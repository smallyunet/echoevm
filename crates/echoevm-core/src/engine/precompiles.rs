//! Fork-aware EVM precompile routing and implementations.

use super::{Halt, gas::words};
use crate::{Fork, kzg::verify_point_evaluation, state::WorldState};
use alloy_primitives::{Address, U256, keccak256};
use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
use num_bigint::BigUint;
use p256::ecdsa::{
    Signature as P256Signature, VerifyingKey as P256VerifyingKey,
    signature::hazmat::PrehashVerifier,
};
use ripemd::Ripemd160;
use sha2::{Digest, Sha256};

pub(super) fn is_precompile(address: Address, fork: Fork) -> bool {
    let bytes = address.as_slice();
    if fork == Fork::Osaka {
        return is_p256verify_precompile(address, fork)
            || (bytes[..19].iter().all(|byte| *byte == 0) && (1..=17).contains(&bytes[19]));
    }
    let maximum = if fork == Fork::Cancun { 10 } else { 17 };
    bytes[..19].iter().all(|byte| *byte == 0) && (1..=maximum).contains(&bytes[19])
}

pub(super) fn warm_precompiles(state: &mut WorldState, fork: Fork) {
    let maximum = if fork == Fork::Cancun { 10 } else { 17 };
    for number in 1..=maximum {
        let mut bytes = [0u8; 20];
        bytes[19] = number;
        state.warm_addresses.insert(Address::from(bytes));
    }
    if fork == Fork::Osaka {
        let mut bytes = [0u8; 20];
        bytes[18..].copy_from_slice(&0x100u16.to_be_bytes());
        state.warm_addresses.insert(Address::from(bytes));
    }
}

fn is_p256verify_precompile(address: Address, fork: Fork) -> bool {
    let bytes = address.as_slice();
    fork == Fork::Osaka && bytes[..18].iter().all(|byte| *byte == 0) && bytes[18..] == [1, 0]
}

pub(super) fn delegation_target(code: &[u8], fork: Fork) -> Option<Address> {
    (fork != Fork::Cancun && code.len() == 23 && code.starts_with(&[0xef, 0x01, 0x00]))
        .then(|| Address::from_slice(&code[3..]))
}

pub(super) fn run_precompile(
    address: Address,
    input: Vec<u8>,
    gas: u64,
    fork: Fork,
) -> (Halt, u64) {
    let bytes = address.as_slice();
    let number = u16::from_be_bytes([bytes[18], bytes[19]]);
    if number == 5
        && fork == Fork::Osaka
        && [0, 32, 64]
            .into_iter()
            .any(|offset| precompile_length(&input, offset) > 1_024)
    {
        return (Halt::Fault("PrecompileError"), 0);
    }
    let cost = match number {
        1 => 3_000,
        2 => 60 + 12 * words(input.len()),
        3 => 600 + 120 * words(input.len()),
        4 => 15 + 3 * words(input.len()),
        5 => modexp_gas(&input, fork),
        6 => 150,
        7 => 6_000,
        8 => 45_000 + 34_000 * (input.len() / 192) as u64,
        9 if input.len() == 213 => u32::from_be_bytes(input[..4].try_into().unwrap()) as u64,
        10 => 50_000,
        11..=17 if fork != Fork::Cancun => crate::bls::gas(number as u8, input.len()).unwrap(),
        0x100 if fork == Fork::Osaka => 6_900,
        _ => return (Halt::Fault("PrecompileError"), 0),
    };
    match gas.checked_sub(cost) {
        Some(remaining) => {
            let output = match number {
                1 => ecrecover_precompile(&input),
                2 => Sha256::digest(input).to_vec(),
                3 => {
                    let hash = Ripemd160::digest(input);
                    let mut output = vec![0; 12];
                    output.extend_from_slice(&hash);
                    output
                }
                4 => input,
                5 => match modexp_precompile(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                6 if input.iter().all(|byte| *byte == 0) => vec![0; 64],
                6 => match crate::bn254::add(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                7 => match crate::bn254::mul(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                8 => match crate::bn254::pairing(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                9 => match blake2f_precompile(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                10 => match verify_point_evaluation(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                11..=17 => match crate::bls::execute(number as u8, &input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                0x100 => p256verify_precompile(&input),
                _ => return (Halt::Fault("PrecompileError"), 0),
            };
            (Halt::Return(output), remaining)
        }
        None => (Halt::Fault("OutOfGas"), 0),
    }
}

pub(super) fn p256verify_precompile(input: &[u8]) -> Vec<u8> {
    if input.len() != 160 {
        return Vec::new();
    }
    let Ok(signature) = P256Signature::from_scalars(
        <[u8; 32]>::try_from(&input[32..64]).unwrap(),
        <[u8; 32]>::try_from(&input[64..96]).unwrap(),
    ) else {
        return Vec::new();
    };
    let mut encoded_key = [0u8; 65];
    encoded_key[0] = 4;
    encoded_key[1..].copy_from_slice(&input[96..160]);
    let Ok(key) = P256VerifyingKey::from_sec1_bytes(&encoded_key) else {
        return Vec::new();
    };
    if key.verify_prehash(&input[..32], &signature).is_err() {
        return Vec::new();
    }
    let mut output = vec![0; 32];
    output[31] = 1;
    output
}

fn blake2f_precompile(input: &[u8]) -> Option<Vec<u8>> {
    const IV: [u64; 8] = [
        0x6a09e667f3bcc908,
        0xbb67ae8584caa73b,
        0x3c6ef372fe94f82b,
        0xa54ff53a5f1d36f1,
        0x510e527fade682d1,
        0x9b05688c2b3e6c1f,
        0x1f83d9abfb41bd6b,
        0x5be0cd19137e2179,
    ];
    const SIGMA: [[usize; 16]; 10] = [
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
        [14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3],
        [11, 8, 12, 0, 5, 2, 15, 13, 10, 14, 3, 6, 7, 1, 9, 4],
        [7, 9, 3, 1, 13, 12, 11, 14, 2, 6, 5, 10, 4, 0, 15, 8],
        [9, 0, 5, 7, 2, 4, 10, 15, 14, 1, 11, 12, 6, 8, 3, 13],
        [2, 12, 6, 10, 0, 11, 8, 3, 4, 13, 7, 5, 15, 14, 1, 9],
        [12, 5, 1, 15, 14, 13, 4, 10, 0, 7, 6, 3, 9, 2, 8, 11],
        [13, 11, 7, 14, 12, 1, 3, 9, 5, 0, 15, 4, 8, 6, 2, 10],
        [6, 15, 14, 9, 11, 3, 0, 8, 12, 2, 13, 7, 1, 4, 10, 5],
        [10, 2, 8, 4, 7, 6, 1, 5, 15, 11, 9, 14, 3, 12, 13, 0],
    ];
    if input.len() != 213 || input[212] > 1 {
        return None;
    }
    let rounds = u32::from_be_bytes(input[..4].try_into().ok()?) as usize;
    let mut h = [0u64; 8];
    let mut m = [0u64; 16];
    for (index, word) in h.iter_mut().enumerate() {
        *word = u64::from_le_bytes(input[4 + index * 8..12 + index * 8].try_into().ok()?);
    }
    for (index, word) in m.iter_mut().enumerate() {
        *word = u64::from_le_bytes(input[68 + index * 8..76 + index * 8].try_into().ok()?);
    }
    let t0 = u64::from_le_bytes(input[196..204].try_into().ok()?);
    let t1 = u64::from_le_bytes(input[204..212].try_into().ok()?);
    let mut v = [0u64; 16];
    v[..8].copy_from_slice(&h);
    v[8..].copy_from_slice(&IV);
    v[12] ^= t0;
    v[13] ^= t1;
    if input[212] == 1 {
        v[14] = !v[14];
    }
    for round in 0..rounds {
        let s = SIGMA[round % 10];
        blake2f_g(&mut v, 0, 4, 8, 12, m[s[0]], m[s[1]]);
        blake2f_g(&mut v, 1, 5, 9, 13, m[s[2]], m[s[3]]);
        blake2f_g(&mut v, 2, 6, 10, 14, m[s[4]], m[s[5]]);
        blake2f_g(&mut v, 3, 7, 11, 15, m[s[6]], m[s[7]]);
        blake2f_g(&mut v, 0, 5, 10, 15, m[s[8]], m[s[9]]);
        blake2f_g(&mut v, 1, 6, 11, 12, m[s[10]], m[s[11]]);
        blake2f_g(&mut v, 2, 7, 8, 13, m[s[12]], m[s[13]]);
        blake2f_g(&mut v, 3, 4, 9, 14, m[s[14]], m[s[15]]);
    }
    let mut output = Vec::with_capacity(64);
    for index in 0..8 {
        output.extend_from_slice(&(h[index] ^ v[index] ^ v[index + 8]).to_le_bytes());
    }
    Some(output)
}

fn blake2f_g(v: &mut [u64; 16], a: usize, b: usize, c: usize, d: usize, x: u64, y: u64) {
    v[a] = v[a].wrapping_add(v[b]).wrapping_add(x);
    v[d] = (v[d] ^ v[a]).rotate_right(32);
    v[c] = v[c].wrapping_add(v[d]);
    v[b] = (v[b] ^ v[c]).rotate_right(24);
    v[a] = v[a].wrapping_add(v[b]).wrapping_add(y);
    v[d] = (v[d] ^ v[a]).rotate_right(16);
    v[c] = v[c].wrapping_add(v[d]);
    v[b] = (v[b] ^ v[c]).rotate_right(63);
}

pub(super) fn ecrecover_precompile(input: &[u8]) -> Vec<u8> {
    let mut padded = [0u8; 128];
    let copy = input.len().min(128);
    padded[..copy].copy_from_slice(&input[..copy]);
    if padded[32..63].iter().any(|byte| *byte != 0) || !matches!(padded[63], 27 | 28) {
        return Vec::new();
    }
    let Ok(mut signature) = Signature::from_scalars(
        <[u8; 32]>::try_from(&padded[64..96]).unwrap(),
        <[u8; 32]>::try_from(&padded[96..128]).unwrap(),
    ) else {
        return Vec::new();
    };
    let Some(mut recovery) = RecoveryId::from_byte(padded[63] - 27) else {
        return Vec::new();
    };
    // Ethereum's precompile accepts both low-s and high-s signatures. k256
    // verifies only low-s signatures, so preserve the recovered key by
    // normalizing s and flipping the recovery y parity together.
    if let Some(normalized) = signature.normalize_s() {
        signature = normalized;
        recovery = RecoveryId::new(!recovery.is_y_odd(), recovery.is_x_reduced());
    }
    let Ok(key) = VerifyingKey::recover_from_prehash(&padded[..32], &signature, recovery) else {
        return Vec::new();
    };
    let encoded = key.to_encoded_point(false);
    let mut output = vec![0; 12];
    output.extend_from_slice(&keccak256(&encoded.as_bytes()[1..])[12..]);
    output
}

pub(super) fn modexp_gas(input: &[u8], fork: Fork) -> u64 {
    let base_len = precompile_length(input, 0);
    let exponent_len = precompile_length(input, 32);
    let modulus_len = precompile_length(input, 64);
    let max_len = base_len.max(modulus_len);
    let words = max_len.saturating_add(7) / 8;
    let complexity = if fork == Fork::Osaka {
        if max_len > 32 {
            2u64.saturating_mul(words.saturating_mul(words))
        } else {
            16
        }
    } else {
        words.saturating_mul(words)
    };
    let head_len = exponent_len.min(32) as usize;
    let base_offset = 96usize.saturating_add(base_len.min(usize::MAX as u64) as usize);
    let mut head = vec![0; head_len];
    if base_offset < input.len() {
        let copy = head_len.min(input.len() - base_offset);
        head[..copy].copy_from_slice(&input[base_offset..base_offset + copy]);
    }
    let head_bits = BigUint::from_bytes_be(&head).bits();
    let iterations = if exponent_len <= 32 {
        head_bits.saturating_sub(1)
    } else {
        (if fork == Fork::Osaka { 16u64 } else { 8u64 })
            .saturating_mul(exponent_len - 32)
            .saturating_add(head_bits.saturating_sub(1))
    }
    .max(1);
    let cost = complexity.saturating_mul(iterations);
    if fork == Fork::Osaka {
        cost.max(500)
    } else {
        cost.saturating_div(3).max(200)
    }
}

fn precompile_length(input: &[u8], offset: usize) -> u64 {
    let mut word = [0u8; 32];
    if offset < input.len() {
        let copy = 32.min(input.len() - offset);
        word[..copy].copy_from_slice(&input[offset..offset + copy]);
    }
    let value = U256::from_be_bytes(word);
    if value > U256::from(u64::MAX) {
        u64::MAX
    } else {
        value.to::<u64>()
    }
}

fn modexp_precompile(input: &[u8]) -> Option<Vec<u8>> {
    let base_len = usize::try_from(precompile_length(input, 0)).ok()?;
    let exponent_len = usize::try_from(precompile_length(input, 32)).ok()?;
    let modulus_len = usize::try_from(precompile_length(input, 64)).ok()?;
    if modulus_len == 0 {
        return Some(Vec::new());
    }
    let base_offset = 96usize;
    let exponent_offset = base_offset.checked_add(base_len)?;
    let modulus_offset = exponent_offset.checked_add(exponent_len)?;
    let base = read_padded_segment(input, base_offset, base_len)?;
    let exponent = read_padded_segment(input, exponent_offset, exponent_len)?;
    let modulus = read_padded_segment(input, modulus_offset, modulus_len)?;
    let modulus = BigUint::from_bytes_be(&modulus);
    if modulus == BigUint::from(0u8) {
        return Some(vec![0; modulus_len]);
    }
    let value = BigUint::from_bytes_be(&base)
        .modpow(&BigUint::from_bytes_be(&exponent), &modulus)
        .to_bytes_be();
    let mut output = vec![0; modulus_len];
    let copy = value.len().min(modulus_len);
    output[modulus_len - copy..].copy_from_slice(&value[value.len() - copy..]);
    Some(output)
}

fn read_padded_segment(input: &[u8], offset: usize, len: usize) -> Option<Vec<u8>> {
    let _ = offset.checked_add(len)?;
    let mut output = vec![0; len];
    if offset < input.len() {
        let copy = len.min(input.len() - offset);
        output[..copy].copy_from_slice(&input[offset..offset + copy]);
    }
    Some(output)
}
