//! EchoEVM's EIP-2537 encoding, validation, gas, and BLS12-381 operations.

use ark_bls12_381::{Bls12_381, Fq, Fq2, Fr, G1Affine, G1Projective, G2Affine, G2Projective};
use ark_ec::{
    AffineRepr, CurveGroup, VariableBaseMSM,
    hashing::{curve_maps::wb::WBMap, map_to_curve_hasher::MapToCurve},
    pairing::Pairing,
};
use ark_ff::{One, PrimeField, Zero};
use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};

const G1_DISCOUNTS: [u16; 128] = [
    1000, 949, 848, 797, 764, 750, 738, 728, 719, 712, 705, 698, 692, 687, 682, 677, 673, 669, 665,
    661, 658, 654, 651, 648, 645, 642, 640, 637, 635, 632, 630, 627, 625, 623, 621, 619, 617, 615,
    613, 611, 609, 608, 606, 604, 603, 601, 599, 598, 596, 595, 593, 592, 591, 589, 588, 586, 585,
    584, 582, 581, 580, 579, 577, 576, 575, 574, 573, 572, 570, 569, 568, 567, 566, 565, 564, 563,
    562, 561, 560, 559, 558, 557, 556, 555, 554, 553, 552, 551, 550, 549, 548, 547, 547, 546, 545,
    544, 543, 542, 541, 540, 540, 539, 538, 537, 536, 536, 535, 534, 533, 532, 532, 531, 530, 529,
    528, 528, 527, 526, 525, 525, 524, 523, 522, 522, 521, 520, 520, 519,
];
const G2_DISCOUNTS: [u16; 128] = [
    1000, 1000, 923, 884, 855, 832, 812, 796, 782, 770, 759, 749, 740, 732, 724, 717, 711, 704,
    699, 693, 688, 683, 679, 674, 670, 666, 663, 659, 655, 652, 649, 646, 643, 640, 637, 634, 632,
    629, 627, 624, 622, 620, 618, 615, 613, 611, 609, 607, 606, 604, 602, 600, 598, 597, 595, 593,
    592, 590, 589, 587, 586, 584, 583, 582, 580, 579, 578, 576, 575, 574, 573, 571, 570, 569, 568,
    567, 566, 565, 563, 562, 561, 560, 559, 558, 557, 556, 555, 554, 553, 552, 552, 551, 550, 549,
    548, 547, 546, 545, 545, 544, 543, 542, 541, 541, 540, 539, 538, 537, 537, 536, 535, 535, 534,
    533, 532, 532, 531, 530, 530, 529, 528, 528, 527, 526, 526, 525, 524, 524,
];

pub(crate) fn gas(address: u8, input_len: usize) -> Option<u64> {
    Some(match address {
        11 => 375,
        12 => msm_gas(input_len / 160, 12_000, &G1_DISCOUNTS),
        13 => 600,
        14 => msm_gas(input_len / 288, 22_500, &G2_DISCOUNTS),
        15 => 37_700 + 32_600 * (input_len / 384) as u64,
        16 => 5_500,
        17 => 23_800,
        _ => return None,
    })
}

fn msm_gas(k: usize, multiplication: u64, discounts: &[u16; 128]) -> u64 {
    if k == 0 {
        return 0;
    }
    let discount = discounts[k.saturating_sub(1).min(127)] as u64;
    k as u64 * multiplication * discount / 1_000
}

pub(crate) fn execute(address: u8, input: &[u8]) -> Option<Vec<u8>> {
    match address {
        11 => g1_add(input),
        12 => g1_msm(input),
        13 => g2_add(input),
        14 => g2_msm(input),
        15 => pairing(input),
        16 => map_fp(input),
        17 => map_fp2(input),
        _ => None,
    }
}

fn g1_add(input: &[u8]) -> Option<Vec<u8>> {
    if input.len() != 256 {
        return None;
    }
    let a = read_g1(&input[..128], false)?;
    let b = read_g1(&input[128..], false)?;
    Some(encode_g1((a.into_group() + b).into_affine()))
}

fn g1_msm(input: &[u8]) -> Option<Vec<u8>> {
    if input.is_empty() || !input.len().is_multiple_of(160) {
        return None;
    }
    let mut points = Vec::new();
    let mut scalars = Vec::new();
    for pair in input.chunks_exact(160) {
        let point = read_g1(&pair[..128], true)?;
        let scalar = Fr::from_be_bytes_mod_order(&pair[128..]);
        if !scalar.is_zero() {
            points.push(point);
            scalars.push(scalar);
        }
    }
    let result = if points.is_empty() {
        G1Projective::zero()
    } else {
        G1Projective::msm(&points, &scalars).ok()?
    };
    Some(encode_g1(result.into_affine()))
}

fn g2_add(input: &[u8]) -> Option<Vec<u8>> {
    if input.len() != 512 {
        return None;
    }
    let a = read_g2(&input[..256], false)?;
    let b = read_g2(&input[256..], false)?;
    Some(encode_g2((a.into_group() + b).into_affine()))
}

fn g2_msm(input: &[u8]) -> Option<Vec<u8>> {
    if input.is_empty() || !input.len().is_multiple_of(288) {
        return None;
    }
    let mut points = Vec::new();
    let mut scalars = Vec::new();
    for pair in input.chunks_exact(288) {
        let point = read_g2(&pair[..256], true)?;
        let scalar = Fr::from_be_bytes_mod_order(&pair[256..]);
        if !scalar.is_zero() {
            points.push(point);
            scalars.push(scalar);
        }
    }
    let result = if points.is_empty() {
        G2Projective::zero()
    } else {
        G2Projective::msm(&points, &scalars).ok()?
    };
    Some(encode_g2(result.into_affine()))
}

fn pairing(input: &[u8]) -> Option<Vec<u8>> {
    if input.is_empty() || !input.len().is_multiple_of(384) {
        return None;
    }
    let mut g1 = Vec::new();
    let mut g2 = Vec::new();
    for pair in input.chunks_exact(384) {
        g1.push(read_g1(&pair[..128], true)?);
        g2.push(read_g2(&pair[128..], true)?);
    }
    let valid = Bls12_381::multi_pairing(g1, g2).0.is_one();
    let mut output = vec![0; 32];
    output[31] = u8::from(valid);
    Some(output)
}

fn map_fp(input: &[u8]) -> Option<Vec<u8>> {
    if input.len() != 64 {
        return None;
    }
    let field = read_fp(input)?;
    let point: G1Affine = WBMap::<ark_bls12_381::g1::Config>::map_to_curve(field)
        .ok()?
        .clear_cofactor();
    Some(encode_g1(point))
}

fn map_fp2(input: &[u8]) -> Option<Vec<u8>> {
    if input.len() != 128 {
        return None;
    }
    let field = Fq2::new(read_fp(&input[..64])?, read_fp(&input[64..])?);
    let point: G2Affine = WBMap::<ark_bls12_381::g2::Config>::map_to_curve(field)
        .ok()?
        .clear_cofactor();
    Some(encode_g2(point))
}

fn read_g1(input: &[u8], subgroup: bool) -> Option<G1Affine> {
    let x = read_fp(input.get(..64)?)?;
    let y = read_fp(input.get(64..128)?)?;
    let point = if x.is_zero() && y.is_zero() {
        G1Affine::zero()
    } else {
        G1Affine::new_unchecked(x, y)
    };
    if !point.is_on_curve() || (subgroup && !point.is_in_correct_subgroup_assuming_on_curve()) {
        None
    } else {
        Some(point)
    }
}

fn read_g2(input: &[u8], subgroup: bool) -> Option<G2Affine> {
    let x = Fq2::new(read_fp(input.get(..64)?)?, read_fp(input.get(64..128)?)?);
    let y = Fq2::new(
        read_fp(input.get(128..192)?)?,
        read_fp(input.get(192..256)?)?,
    );
    let point = if x.is_zero() && y.is_zero() {
        G2Affine::zero()
    } else {
        G2Affine::new_unchecked(x, y)
    };
    if !point.is_on_curve() || (subgroup && !point.is_in_correct_subgroup_assuming_on_curve()) {
        None
    } else {
        Some(point)
    }
}

fn read_fp(input: &[u8]) -> Option<Fq> {
    if input.len() != 64 || input[..16].iter().any(|byte| *byte != 0) {
        return None;
    }
    let mut little_endian = input[16..].to_vec();
    little_endian.reverse();
    Fq::deserialize_uncompressed(little_endian.as_slice()).ok()
}

fn encode_fp(value: Fq) -> [u8; 64] {
    let mut raw = [0u8; 48];
    value
        .serialize_uncompressed(raw.as_mut_slice())
        .expect("field serialization");
    raw.reverse();
    let mut output = [0u8; 64];
    output[16..].copy_from_slice(&raw);
    output
}

fn encode_g1(point: G1Affine) -> Vec<u8> {
    let mut output = vec![0; 128];
    if let Some((x, y)) = point.xy() {
        output[..64].copy_from_slice(&encode_fp(x));
        output[64..].copy_from_slice(&encode_fp(y));
    }
    output
}

fn encode_g2(point: G2Affine) -> Vec<u8> {
    let mut output = vec![0; 256];
    if let Some((x, y)) = point.xy() {
        output[..64].copy_from_slice(&encode_fp(x.c0));
        output[64..128].copy_from_slice(&encode_fp(x.c1));
        output[128..192].copy_from_slice(&encode_fp(y.c0));
        output[192..].copy_from_slice(&encode_fp(y.c1));
    }
    output
}
