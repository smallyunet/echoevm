use super::*;
use echoevm_protocol::{ReplayWitness, WITNESS_SCHEMA};

pub struct TransactionWitnessMaterialization<'a> {
    pub chain_id: u64,
    pub block_hash: B256,
    pub header: serde_json::Value,
    pub transactions: &'a [alloy_primitives::Bytes],
    pub target_index: usize,
    pub parent_prestate: &'a BTreeMap<Address, WitnessAccount>,
    pub block_hashes: BTreeMap<String, B256>,
    pub source: Option<String>,
}

pub fn materialize_transaction_witness(
    input: TransactionWitnessMaterialization<'_>,
) -> Result<ReplayWitness, ExecuteError> {
    let TransactionWitnessMaterialization {
        chain_id,
        block_hash,
        header: header_value,
        transactions,
        target_index,
        parent_prestate,
        block_hashes,
        source,
    } = input;
    let header: Header = serde_json::from_value(header_value.clone())
        .map_err(|error| ExecuteError::Witness(format!("decode header: {error}")))?;
    if header.hash_slow() != block_hash {
        return Err(ExecuteError::Witness(
            "blockHash does not match header hash".into(),
        ));
    }
    let envelopes = decode_envelopes(transactions)?;
    envelopes.get(target_index).ok_or_else(|| {
        ExecuteError::Witness(format!("transaction index {target_index} is outside block"))
    })?;
    let decoded_hashes = decode_block_hashes(&block_hashes)?;
    let fork = Fork::for_timestamp(header.timestamp);
    let environment =
        block_environment(chain_id, &header, decoded_hashes.clone(), Vec::new(), fork);
    let mut world = world_from_prestate(parent_prestate, true);
    apply_pre_block_system_calls(&mut world, &header, fork, &environment)?;
    fail_on_missing(&world)?;

    let mut target_world = None;
    for (index, envelope) in envelopes.iter().take(target_index + 1).enumerate() {
        if index == target_index {
            target_world = Some(world.clone());
        }
        let transaction = transaction_from_envelope(
            envelope,
            chain_id,
            &header,
            decoded_hashes.clone(),
            fork,
            false,
        )?;
        engine::transact(&mut world, transaction)
            .map_err(|error| ExecuteError::Evm(format!("transaction {index}: {error}")))?;
        fail_on_missing(&world)?;
    }
    let target_world = target_world.expect("target world captured");
    Ok(ReplayWitness {
        schema: WITNESS_SCHEMA.into(),
        chain_id,
        block_hash,
        transaction_index: target_index as u64,
        header: header_value,
        transaction: transactions[target_index].clone(),
        prestate: witness_prestate(&target_world),
        block_hashes,
        source,
    })
}

fn witness_prestate(world: &WorldState) -> BTreeMap<Address, WitnessAccount> {
    world
        .accounts
        .keys()
        .chain(world.known_accounts())
        .copied()
        .collect::<std::collections::BTreeSet<_>>()
        .into_iter()
        .map(|address| {
            let Some(account) = world.accounts.get(&address) else {
                return (
                    address,
                    WitnessAccount {
                        exists: Some(false),
                        storage_complete: true,
                        ..Default::default()
                    },
                );
            };
            let storage = world
                .known_storage_slots()
                .iter()
                .filter(|(known_address, _)| known_address == &address)
                .map(|(_, slot)| {
                    let value = account.storage.get(slot).copied().unwrap_or_default();
                    (
                        B256::from(slot.to_be_bytes::<32>()),
                        B256::from(value.to_be_bytes::<32>()),
                    )
                })
                .collect();
            (
                address,
                WitnessAccount {
                    exists: Some(true),
                    balance: Some(account.balance),
                    nonce: account.nonce,
                    code: account.code.clone().into(),
                    storage,
                    storage_complete: world.is_storage_complete(address),
                },
            )
        })
        .collect()
}
