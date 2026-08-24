use super::*;

pub(super) fn apply_authorizations(state: &mut WorldState, transaction: &Transaction) {
    for authorization in &transaction.authorization_list {
        if !(authorization.chain_id.is_zero()
            || authorization.chain_id == U256::from(transaction.environment.chain_id))
            || authorization.nonce == u64::MAX
        {
            continue;
        }
        let Some(authority) = recover_authority(authorization) else {
            continue;
        };
        state.warm_addresses.insert(authority);
        let account = state.account(authority).cloned().unwrap_or_default();
        let delegated = account.code.len() == 23 && account.code.starts_with(&[0xef, 0x01, 0x00]);
        if (!account.code.is_empty() && !delegated) || account.nonce != authorization.nonce {
            continue;
        }
        if !account.is_empty() {
            state.refund += 12_500;
        }
        let account = state.account_mut(authority);
        account.code = if authorization.delegate.is_zero() {
            Vec::new()
        } else {
            let mut code = vec![0xef, 0x01, 0x00];
            code.extend_from_slice(authorization.delegate.as_slice());
            code
        };
        account.nonce = account.nonce.saturating_add(1);
    }
}

pub(super) fn recover_authority(authorization: &Authorization) -> Option<Address> {
    if authorization.y_parity > 1 {
        return None;
    }
    let signature = Signature::from_scalars(
        authorization.r.to_be_bytes::<32>(),
        authorization.s.to_be_bytes::<32>(),
    )
    .ok()?;
    if signature.normalize_s().is_some() {
        return None;
    }
    let mut payload = Vec::new();
    rlp_push_quantity(&mut payload, authorization.chain_id);
    payload.push(0x94);
    payload.extend_from_slice(authorization.delegate.as_slice());
    rlp_push_quantity(&mut payload, U256::from(authorization.nonce));
    let mut message = Vec::with_capacity(payload.len() + 2);
    message.push(0x05);
    rlp_push_list_header(&mut message, payload.len());
    message.extend(payload);
    let hash = keccak256(message);
    let key = VerifyingKey::recover_from_prehash(
        hash.as_slice(),
        &signature,
        RecoveryId::from_byte(authorization.y_parity)?,
    )
    .ok()?;
    let encoded = key.to_encoded_point(false);
    Some(Address::from_slice(
        &keccak256(&encoded.as_bytes()[1..])[12..],
    ))
}

pub(super) fn rlp_push_quantity(output: &mut Vec<u8>, value: U256) {
    if value.is_zero() {
        output.push(0x80);
        return;
    }
    let bytes = value.to_be_bytes::<32>();
    let first = bytes.iter().position(|byte| *byte != 0).unwrap();
    let value = &bytes[first..];
    if value.len() == 1 && value[0] < 0x80 {
        output.push(value[0]);
    } else {
        output.push(0x80 + value.len() as u8);
        output.extend_from_slice(value);
    }
}

pub(super) fn rlp_push_list_header(output: &mut Vec<u8>, len: usize) {
    if len < 56 {
        output.push(0xc0 + len as u8);
    } else {
        let bytes = len.to_be_bytes();
        let first = bytes.iter().position(|byte| *byte != 0).unwrap();
        let encoded = &bytes[first..];
        output.push(0xf7 + encoded.len() as u8);
        output.extend_from_slice(encoded);
    }
}
