use super::*;

pub fn replay_witness(
    witness: &ReplayWitness,
    include_trace: bool,
) -> Result<ReplayResult, ExecuteError> {
    witness
        .validate()
        .map_err(|error| ExecuteError::Witness(error.to_string()))?;
    let header: Header = serde_json::from_value(witness.header.clone())
        .map_err(|error| ExecuteError::Witness(format!("decode header: {error}")))?;
    let computed_hash = header.hash_slow();
    if computed_hash != witness.block_hash {
        return Err(ExecuteError::Witness(format!(
            "blockHash {} does not match header hash {}",
            witness.block_hash, computed_hash
        )));
    }

    let mut raw = witness.transaction.as_ref();
    let envelope = TxEnvelope::decode_2718(&mut raw)
        .map_err(|error| ExecuteError::Witness(format!("decode transaction: {error}")))?;
    if !raw.is_empty() {
        return Err(ExecuteError::Witness(
            "transaction has trailing bytes".into(),
        ));
    }
    if let Some(chain_id) = envelope.chain_id()
        && chain_id != witness.chain_id
    {
        return Err(ExecuteError::Witness(format!(
            "transaction chainId {chain_id} does not match witness chainId {}",
            witness.chain_id
        )));
    }
    let caller = envelope
        .recover_signer()
        .map_err(|error| ExecuteError::Witness(format!("recover transaction sender: {error}")))?;
    if !witness.prestate.contains_key(&caller) {
        return Err(ExecuteError::Witness(format!(
            "replay witness is missing sender account {caller}"
        )));
    }

    let mut world = crate::block::world_from_prestate(&witness.prestate, true);
    let mut block_hashes = BTreeMap::new();
    for (number, hash) in &witness.block_hashes {
        let number = number.parse::<u64>().map_err(|error| {
            ExecuteError::Witness(format!("invalid blockHashes key {number:?}: {error}"))
        })?;
        block_hashes.insert(number, *hash);
    }

    let fork = Fork::for_timestamp(header.timestamp);
    let transaction = crate::block::transaction_from_envelope(
        &envelope,
        witness.chain_id,
        &header,
        block_hashes,
        fork,
        include_trace,
    )?;
    let mut execution = engine::transact(&mut world, transaction)
        .map_err(|error| ExecuteError::Evm(error.into()))?;
    let missing = world.missing_reads();
    if !missing.accounts.is_empty() || !missing.storage.is_empty() {
        return Err(ExecuteError::IncompleteWitness {
            accounts: missing.accounts.into_iter().collect(),
            storage: missing
                .storage
                .into_iter()
                .map(|(address, slot)| (address, B256::from(slot.to_be_bytes::<32>())))
                .collect(),
        });
    }
    let gas_used = execution.gas_used;
    let status = match &execution.status {
        ExecutionStatus::Success => "success",
        ExecutionStatus::Revert => "revert",
        ExecutionStatus::Fault => "fault",
    };
    let state = flatten_world_state(&world);
    execution.storage = state.clone();

    let canonical = serde_json::to_vec(witness)
        .map_err(|error| ExecuteError::Witness(format!("encode provenance: {error}")))?;
    let digest = Sha256::digest(canonical);
    let tx_hash = *envelope.tx_hash();
    let evidence = include_trace.then(|| build_evidence(&execution, "auto", 40));
    Ok(ReplayResult {
        transaction: TransactionSummary {
            hash: tx_hash.to_string(),
            explorer_url: format!("https://etherscan.io/tx/{tx_hash}"),
            chain_id: witness.chain_id,
            block_number: header.number,
            block_hash: witness.block_hash.to_string(),
            transaction_index: witness.transaction_index,
            from: caller.to_string(),
            to: envelope.to().map(|address| address.to_string()),
            value: envelope.value().to_string(),
            gas_limit: envelope.gas_limit(),
            gas_used,
            transaction_type: envelope.ty(),
            input: format!("0x{}", hex::encode(envelope.input())),
            status: status.into(),
            fork: fork.name().into(),
        },
        execution,
        state,
        warnings: replay_warnings(header.timestamp),
        witness: WitnessProvenance {
            schema: WITNESS_SCHEMA.into(),
            sha256: hex::encode(digest),
            source: witness.source.clone(),
        },
        evidence,
    })
}

fn flatten_world_state(world: &state::WorldState) -> BTreeMap<String, String> {
    let mut state = BTreeMap::new();
    for (address, account) in &world.accounts {
        let prefix = address.to_string().to_lowercase();
        state.insert(format!("{prefix}:balance"), account.balance.to_string());
        state.insert(format!("{prefix}:nonce"), account.nonce.to_string());
        if !account.code.is_empty() {
            state.insert(
                format!("{prefix}:code"),
                format!("0x{}", hex::encode(&account.code)),
            );
        }
        for (slot, value) in &account.storage {
            state.insert(
                format!("{prefix}:storage:0x{slot:064x}"),
                format!("0x{value:064x}"),
            );
        }
    }
    state
}

fn replay_warnings(timestamp: u64) -> Vec<String> {
    if timestamp < MAINNET_CANCUN_TIME {
        vec!["EchoEVM v1 Mainnet replay supports Cancun through Osaka; this pre-Cancun witness is executed with the Cancun baseline.".into()]
    } else {
        Vec::new()
    }
}
