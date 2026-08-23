//! EchoEVM implementation of the EIP-196/EIP-197 BN254 precompile ABI.

use ark_bn254::{Bn254, Fq, Fq2, Fr, G1Affine, G2Affine};
use ark_ec::{AffineRepr, CurveGroup, pairing::Pairing};
use ark_ff::{BigInteger, One, PrimeField, Zero};

pub(crate) fn add(input: &[u8]) -> Option<Vec<u8>> {
    let input = padded(input, 128);
    let a = read_g1(&input[..64])?;
    let b = read_g1(&input[64..])?;
    Some(encode_g1((a.into_group() + b).into_affine()))
}

pub(crate) fn mul(input: &[u8]) -> Option<Vec<u8>> {
    let input = padded(input, 96);
    let point = read_g1(&input[..64])?;
    let scalar = Fr::from_be_bytes_mod_order(&input[64..]);
    Some(encode_g1(
        point.mul_bigint(scalar.into_bigint()).into_affine(),
    ))
}

pub(crate) fn pairing(input: &[u8]) -> Option<Vec<u8>> {
    if !input.len().is_multiple_of(192) {
        return None;
    }
    let mut g1 = Vec::new();
    let mut g2 = Vec::new();
    for pair in input.chunks_exact(192) {
        g1.push(read_g1(&pair[..64])?);
        g2.push(read_g2(&pair[64..])?);
    }
    let valid = Bn254::multi_pairing(g1, g2).0.is_one();
    let mut output = vec![0; 32];
    output[31] = u8::from(valid);
    Some(output)
}

fn read_g1(input: &[u8]) -> Option<G1Affine> {
    let x = read_fq(&input[..32])?;
    let y = read_fq(&input[32..64])?;
    let point = if x.is_zero() && y.is_zero() {
        G1Affine::zero()
    } else {
        G1Affine::new_unchecked(x, y)
    };
    point.is_on_curve().then_some(point)
}

fn read_g2(input: &[u8]) -> Option<G2Affine> {
    let x = Fq2::new(read_fq(&input[32..64])?, read_fq(&input[..32])?);
    let y = Fq2::new(read_fq(&input[96..128])?, read_fq(&input[64..96])?);
    let point = if x.is_zero() && y.is_zero() {
        G2Affine::zero()
    } else {
        G2Affine::new_unchecked(x, y)
    };
    (point.is_on_curve() && point.is_in_correct_subgroup_assuming_on_curve()).then_some(point)
}

fn read_fq(input: &[u8]) -> Option<Fq> {
    let value = Fq::from_be_bytes_mod_order(input);
    (encode_fq(value).as_slice() == input).then_some(value)
}

fn encode_g1(point: G1Affine) -> Vec<u8> {
    let mut output = vec![0; 64];
    if let Some((x, y)) = point.xy() {
        output[..32].copy_from_slice(&encode_fq(x));
        output[32..].copy_from_slice(&encode_fq(y));
    }
    output
}

fn encode_fq(value: Fq) -> [u8; 32] {
    let bytes = value.into_bigint().to_bytes_be();
    let mut output = [0; 32];
    output[32 - bytes.len()..].copy_from_slice(&bytes);
    output
}

fn padded(input: &[u8], len: usize) -> Vec<u8> {
    let mut output = vec![0; len];
    let copy = input.len().min(len);
    output[..copy].copy_from_slice(&input[..copy]);
    output
}
