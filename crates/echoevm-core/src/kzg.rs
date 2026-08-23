use ark_bls12_381::{Bls12_381, Fr, G1Affine, G2Affine};
use ark_ec::{AffineRepr, CurveGroup, pairing::Pairing};
use ark_ff::PrimeField;
use ark_serialize::CanonicalDeserialize;
use sha2::{Digest, Sha256};

const BLS_MODULUS: [u8; 32] = [
    0x73, 0xed, 0xa7, 0x53, 0x29, 0x9d, 0x7d, 0x48, 0x33, 0x39, 0xd8, 0x08, 0x09, 0xa1, 0xd8, 0x05,
    0x53, 0xbd, 0xa4, 0x02, 0xff, 0xfe, 0x5b, 0xfe, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x01,
];

// The first two G2 monomial points from Ethereum's canonical KZG ceremony setup:
// [1]₂ and [s]₂. A single point-evaluation proof needs no other setup points.
const G2_GENERATOR: &str = "93e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8";
const G2_S: &str = "b5bfd7dd8cdeb128843bc287230af38926187075cbfbefa81009a2ce615ac53d2914e5870cb452d2afaaab24f3499f72185cbfee53492714734429b7b38608e23926c911cceceac9a36851477ba4c60b087041de621000edc98edada20c1def2";

pub(crate) fn verify_point_evaluation(input: &[u8]) -> Option<Vec<u8>> {
    if input.len() != 192 || input[0] != 1 {
        return None;
    }

    let commitment = &input[96..144];
    let digest = Sha256::digest(commitment);
    if input[1..32] != digest[1..] {
        return None;
    }

    let z_bytes: &[u8; 32] = input[32..64].try_into().ok()?;
    let y_bytes: &[u8; 32] = input[64..96].try_into().ok()?;
    if z_bytes >= &BLS_MODULUS || y_bytes >= &BLS_MODULUS {
        return None;
    }

    let commitment = G1Affine::deserialize_compressed(commitment).ok()?;
    let proof = G1Affine::deserialize_compressed(&input[144..192]).ok()?;
    let g2 = G2Affine::deserialize_compressed(hex::decode(G2_GENERATOR).ok()?.as_slice()).ok()?;
    let s_g2 = G2Affine::deserialize_compressed(hex::decode(G2_S).ok()?.as_slice()).ok()?;
    let z = Fr::from_be_bytes_mod_order(z_bytes);
    let y = Fr::from_be_bytes_mod_order(y_bytes);

    let left = commitment.into_group() - G1Affine::generator().mul_bigint(y.into_bigint());
    let right = s_g2.into_group() - g2.mul_bigint(z.into_bigint());
    if Bls12_381::pairing(left.into_affine(), g2) != Bls12_381::pairing(proof, right.into_affine())
    {
        return None;
    }

    let mut output = vec![0; 64];
    output[30] = 0x10;
    output[32..].copy_from_slice(&BLS_MODULUS);
    Some(output)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_wrong_input_length() {
        assert!(verify_point_evaluation(&[]).is_none());
    }
}
