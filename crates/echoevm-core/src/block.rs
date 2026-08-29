use crate::{
    ENGINE_NAME, ENGINE_VERSION, ExecuteError, Fork,
    engine::{self, Authorization, Environment, Transaction},
    state::WorldState,
};
use alloy_consensus::{
    Eip658Value, Header, Receipt, ReceiptEnvelope, Transaction as AlloyTransaction, TxEnvelope,
    proofs::{calculate_receipt_root, calculate_transaction_root, calculate_withdrawals_root},
    transaction::SignerRecoverable,
};
use alloy_eips::{Decodable2718, eip4895::Withdrawal};
use alloy_primitives::{Address, B256, Bloom, U256};
use echoevm_protocol::{
    BLOCK_RESULT_SCHEMA, BlockExecutionResult, BlockWitness, ExecutionStatus, WitnessAccount,
};
use std::collections::BTreeMap;

mod materialize;
mod system;

pub use materialize::{TransactionWitnessMaterialization, materialize_transaction_witness};
use system::{apply_post_block_system_calls, apply_pre_block_system_calls};

pub fn execute_block_witness(
    witness: &BlockWitness,
    trace_transaction: Option<usize>,
) -> Result<BlockExecutionResult, ExecuteError> {
    witness
        .validate()
        .map_err(|error| ExecuteError::Witness(error.to_string()))?;
    let header: Header = serde_json::from_value(witness.header.clone())
        .map_err(|error| ExecuteError::Witness(format!("decode header: {error}")))?;
    if header.hash_slow() != witness.block_hash {
        return Err(ExecuteError::Witness(
            "blockHash does not match header hash".into(),
        ));
    }
    let envelopes = decode_envelopes(&witness.transactions)?;
    if trace_transaction.is_some_and(|index| index >= envelopes.len()) {
        return Err(ExecuteError::Witness(
            "trace transaction index is outside the block".into(),
        ));
    }
    if calculate_transaction_root(&envelopes) != header.transactions_root {
        return Err(ExecuteError::Witness(
            "transactions do not match header transactionsRoot".into(),
        ));
    }
    let withdrawals = witness
        .withdrawals
        .iter()
        .map(|item| Withdrawal {
            index: item.index,
            validator_index: item.validator_index,
            address: item.address,
            amount: item.amount,
        })
        .collect::<Vec<_>>();
    if let Some(expected) = header.withdrawals_root
        && calculate_withdrawals_root(&withdrawals) != expected
    {
        return Err(ExecuteError::Witness(
            "withdrawals do not match header withdrawalsRoot".into(),
        ));
    }

    let mut world = world_from_prestate(&witness.prestate, true);
    let block_hashes = decode_block_hashes(&witness.block_hashes)?;
    let fork = declared_fork(witness.fork);
    let environment = block_environment(witness.chain_id, &header, block_hashes, Vec::new(), fork);
    apply_pre_block_system_calls(&mut world, &header, fork, &environment)?;

    let mut cumulative_gas = 0u64;
    let mut executions = Vec::with_capacity(envelopes.len());
    let mut receipts = Vec::with_capacity(envelopes.len());
    let mut all_logs = Vec::new();
    for (index, envelope) in envelopes.iter().enumerate() {
        let transaction = transaction_from_envelope(
            envelope,
            witness.chain_id,
            &header,
            environment.block_hashes.clone(),
            fork,
            trace_transaction == Some(index),
        )?;
        let result = engine::transact(&mut world, transaction)
            .map_err(|error| ExecuteError::Evm(format!("transaction {index}: {error}")))?;
        cumulative_gas = cumulative_gas
            .checked_add(result.gas_used)
            .ok_or_else(|| ExecuteError::Evm("block gas overflow".into()))?;
        if cumulative_gas > header.gas_limit {
            return Err(ExecuteError::Evm("block gas limit exceeded".into()));
        }
        let receipt = Receipt {
            status: Eip658Value::Eip658(result.status == ExecutionStatus::Success),
            cumulative_gas_used: cumulative_gas,
            logs: world.logs.clone(),
        };
        all_logs.extend(receipt.logs.iter().cloned());
        receipts.push(ReceiptEnvelope::from_typed(
            envelope.tx_type(),
            receipt.with_bloom(),
        ));
        executions.push(result);
    }
    apply_post_block_system_calls(&mut world, fork, &environment)?;
    for withdrawal in &withdrawals {
        let balance = world.balance(withdrawal.address);
        let account = world.account_mut(withdrawal.address);
        account.balance = balance.wrapping_add(withdrawal.amount_wei());
    }
    fail_on_missing(&world)?;

    let state_root = world.state_root();
    let receipts_root = calculate_receipt_root(&receipts);
    let logs_bloom: Bloom = all_logs.iter().collect();
    assert_header_commitments(
        &header,
        cumulative_gas,
        state_root,
        receipts_root,
        logs_bloom,
        &executions
            .iter()
            .map(|execution| {
                (
                    execution.gas_used,
                    execution.status.clone(),
                    execution.error.clone(),
                )
            })
            .collect::<Vec<_>>(),
    )?;
    Ok(BlockExecutionResult {
        schema: BLOCK_RESULT_SCHEMA.into(),
        engine: ENGINE_NAME.into(),
        engine_version: ENGINE_VERSION.into(),
        block_hash: witness.block_hash.to_string(),
        block_number: header.number,
        fork: fork.name().into(),
        transaction_count: executions.len(),
        gas_used: cumulative_gas,
        state_root: state_root.to_string(),
        receipts_root: receipts_root.to_string(),
        logs_bloom: logs_bloom.to_string(),
        transactions: executions,
        warnings: (fork != Fork::Cancun)
            .then(|| "Prague/Osaka request system calls are applied to state, but requestsHash is not independently recomputed from the emitted request list.".into())
            .into_iter()
            .collect(),
    })
}

fn fail_on_missing(world: &WorldState) -> Result<(), ExecuteError> {
    let missing = world.missing_reads();
    if missing.accounts.is_empty() && missing.storage.is_empty() {
        return Ok(());
    }
    Err(ExecuteError::IncompleteWitness {
        accounts: missing.accounts.into_iter().collect(),
        storage: missing
            .storage
            .into_iter()
            .map(|(address, slot)| (address, B256::from(slot.to_be_bytes::<32>())))
            .collect(),
    })
}

fn decode_envelopes(
    transactions: &[alloy_primitives::Bytes],
) -> Result<Vec<TxEnvelope>, ExecuteError> {
    transactions
        .iter()
        .enumerate()
        .map(|(index, bytes)| {
            let mut raw = bytes.as_ref();
            let envelope = TxEnvelope::decode_2718(&mut raw).map_err(|error| {
                ExecuteError::Witness(format!("decode transaction {index}: {error}"))
            })?;
            if !raw.is_empty() {
                return Err(ExecuteError::Witness(format!(
                    "transaction {index} has trailing bytes"
                )));
            }
            Ok(envelope)
        })
        .collect()
}

fn declared_fork(fork: echoevm_protocol::TestFork) -> Fork {
    match fork {
        echoevm_protocol::TestFork::Cancun => Fork::Cancun,
        echoevm_protocol::TestFork::Prague => Fork::Prague,
        echoevm_protocol::TestFork::Osaka => Fork::Osaka,
    }
}

pub(crate) fn transaction_from_envelope(
    envelope: &TxEnvelope,
    chain_id: u64,
    header: &Header,
    block_hashes: BTreeMap<u64, B256>,
    fork: Fork,
    trace: bool,
) -> Result<Transaction, ExecuteError> {
    if envelope.chain_id().is_some_and(|actual| actual != chain_id) {
        return Err(ExecuteError::Witness("transaction chainId mismatch".into()));
    }
    let caller = envelope
        .recover_signer()
        .map_err(|error| ExecuteError::Witness(format!("recover transaction sender: {error}")))?;
    let authorizations = envelope
        .authorization_list()
        .unwrap_or_default()
        .iter()
        .map(|item| Authorization {
            chain_id: *item.chain_id(),
            delegate: *item.address(),
            nonce: item.nonce(),
            y_parity: item.y_parity(),
            r: item.r(),
            s: item.s(),
        })
        .collect::<Vec<_>>();
    let environment = block_environment(
        chain_id,
        header,
        block_hashes,
        envelope
            .blob_versioned_hashes()
            .unwrap_or_default()
            .to_vec(),
        fork,
    );
    Ok(Transaction {
        caller,
        to: envelope.to(),
        value: envelope.value(),
        data: envelope.input().to_vec(),
        gas_limit: envelope.gas_limit(),
        gas_price: U256::from(envelope.effective_gas_price(header.base_fee_per_gas)),
        max_fee_per_gas: U256::from(envelope.max_fee_per_gas()),
        max_priority_fee_per_gas: envelope.max_priority_fee_per_gas().map(U256::from),
        nonce: envelope.nonce(),
        access_list: envelope
            .access_list()
            .map(|list| {
                list.0
                    .iter()
                    .map(|item| {
                        (
                            item.address,
                            item.storage_keys
                                .iter()
                                .map(|key| U256::from_be_bytes(key.0))
                                .collect(),
                        )
                    })
                    .collect()
            })
            .unwrap_or_default(),
        authorization_list: authorizations,
        set_code: envelope.is_eip7702(),
        max_fee_per_blob_gas: envelope.max_fee_per_blob_gas().map(U256::from),
        fork,
        environment,
        trace,
    })
}

pub(crate) fn world_from_prestate(
    prestate: &BTreeMap<Address, WitnessAccount>,
    track_missing: bool,
) -> WorldState {
    let mut world = WorldState::default();
    if track_missing {
        world.enable_missing_tracking();
    }
    for (address, account) in prestate {
        if account.exists == Some(false) {
            world.mark_known_absent_account(*address);
            continue;
        }
        world.mark_known_account(*address);
        if account.storage_complete {
            world.mark_storage_complete(*address);
        }
        for slot in account.storage.keys() {
            world.mark_known_storage(*address, U256::from_be_bytes(slot.0));
        }
        let target = world.account_mut(*address);
        target.balance = account.balance.unwrap_or_default();
        target.nonce = account.nonce;
        target.code = account.code.to_vec();
        target.storage = account
            .storage
            .iter()
            .filter(|(_, value)| **value != B256::ZERO)
            .map(|(slot, value)| (U256::from_be_bytes(slot.0), U256::from_be_bytes(value.0)))
            .collect();
    }
    world
}

fn decode_block_hashes(
    input: &BTreeMap<String, B256>,
) -> Result<BTreeMap<u64, B256>, ExecuteError> {
    input
        .iter()
        .map(|(number, hash)| {
            number
                .parse::<u64>()
                .map(|number| (number, *hash))
                .map_err(|error| ExecuteError::Witness(format!("invalid blockHashes key: {error}")))
        })
        .collect()
}

fn block_environment(
    chain_id: u64,
    header: &Header,
    block_hashes: BTreeMap<u64, B256>,
    blob_hashes: Vec<B256>,
    fork: Fork,
) -> Environment {
    Environment {
        chain_id,
        block_number: header.number,
        timestamp: header.timestamp,
        coinbase: header.beneficiary,
        block_gas_limit: header.gas_limit,
        base_fee: U256::from(header.base_fee_per_gas.unwrap_or_default()),
        prevrandao: U256::from_be_bytes(header.mix_hash.0),
        blob_base_fee: fake_exponential(header.excess_blob_gas.unwrap_or_default(), fork),
        block_hashes,
        blob_hashes,
    }
}

fn fake_exponential(excess_blob_gas: u64, fork: Fork) -> U256 {
    let denominator = if fork == Fork::Cancun {
        3_338_477u64
    } else {
        5_007_716u64
    };
    let mut output = U256::ZERO;
    let mut accumulator = U256::from(denominator);
    let mut index = 1u64;
    while !accumulator.is_zero() {
        output += accumulator;
        accumulator = accumulator * U256::from(excess_blob_gas)
            / U256::from(denominator.saturating_mul(index));
        index += 1;
    }
    output / U256::from(denominator)
}

fn assert_header_commitments(
    header: &Header,
    gas_used: u64,
    state_root: B256,
    receipts_root: B256,
    logs_bloom: Bloom,
    transactions: &[(u64, ExecutionStatus, Option<String>)],
) -> Result<(), ExecuteError> {
    let mismatch = if gas_used != header.gas_used {
        Some(format!(
            "gasUsed {gas_used} != {} (transactions {transactions:?})",
            header.gas_used
        ))
    } else if state_root != header.state_root {
        Some(format!("stateRoot {state_root} != {}", header.state_root))
    } else if receipts_root != header.receipts_root {
        Some(format!(
            "receiptsRoot {receipts_root} != {}",
            header.receipts_root
        ))
    } else if logs_bloom != header.logs_bloom {
        Some("logsBloom mismatch".into())
    } else {
        None
    };
    mismatch
        .map(|message| {
            Err(ExecuteError::Evm(format!(
                "block commitment mismatch: {message}"
            )))
        })
        .unwrap_or(Ok(()))
}

#[cfg(test)]
#[path = "block/tests.rs"]
mod tests;
