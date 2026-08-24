use super::*;

pub(super) fn create_address(caller: Address, nonce: u64) -> Address {
    let mut payload = Vec::with_capacity(31);
    payload.push(0x94);
    payload.extend_from_slice(caller.as_slice());
    match nonce {
        0 => payload.push(0x80),
        1..=0x7f => payload.push(nonce as u8),
        _ => {
            let bytes = nonce.to_be_bytes();
            let first = bytes.iter().position(|byte| *byte != 0).unwrap_or(7);
            let encoded = &bytes[first..];
            payload.push(0x80 + encoded.len() as u8);
            payload.extend_from_slice(encoded);
        }
    }
    let mut encoded = Vec::with_capacity(payload.len() + 1);
    encoded.push(0xc0 + payload.len() as u8);
    encoded.extend(payload);
    Address::from_slice(&keccak256(encoded)[12..])
}

pub(super) fn create2_address(caller: Address, salt: U256, initcode: &[u8]) -> Address {
    let mut input = Vec::with_capacity(85);
    input.push(0xff);
    input.extend_from_slice(caller.as_slice());
    input.extend_from_slice(&salt.to_be_bytes::<32>());
    input.extend_from_slice(keccak256(initcode).as_slice());
    Address::from_slice(&keccak256(input)[12..])
}

pub(super) fn address_word(address: Address) -> U256 {
    U256::from_be_slice(address.as_slice())
}

pub(super) fn word_address(value: U256) -> Address {
    let bytes = value.to_be_bytes::<32>();
    Address::from_slice(&bytes[12..])
}
