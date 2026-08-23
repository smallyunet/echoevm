//! Independent EchoEVM execution kernel.

mod bls;
mod bn254;
mod engine;
mod kzg;
pub mod opcode;
pub mod state;

pub use engine::{Authorization, Environment as BlockEnv, Transaction, transact};

use alloy_consensus::{
    Header, Transaction as AlloyTransaction, TxEnvelope, transaction::SignerRecoverable,
};
use alloy_eips::{Decodable2718, Typed2718};
use alloy_primitives::{Address, B256, U256};
use echoevm_protocol::{
    ExecutionResult as WireResult, ExecutionStatus, ReplayResult, ReplayWitness, TraceStep,
    TransactionSummary, WITNESS_SCHEMA, WitnessProvenance,
};
use serde_json::json;
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;

pub const ENGINE_NAME: &str = "EchoEVM";
pub const ENGINE_VERSION: &str = env!("CARGO_PKG_VERSION");
pub const DEFAULT_GAS_LIMIT: u64 = 15_000_000;
pub const MAINNET_CANCUN_TIME: u64 = 1_710_338_135;
pub const MAINNET_PRAGUE_TIME: u64 = 1_746_612_311;
pub const MAINNET_OSAKA_TIME: u64 = 1_764_798_551;

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum Fork {
    Cancun,
    Prague,
    #[default]
    Osaka,
}

impl Fork {
    pub const fn for_timestamp(timestamp: u64) -> Self {
        if timestamp >= MAINNET_OSAKA_TIME {
            Self::Osaka
        } else if timestamp >= MAINNET_PRAGUE_TIME {
            Self::Prague
        } else {
            Self::Cancun
        }
    }

    pub const fn name(self) -> &'static str {
        match self {
            Self::Cancun => "Cancun",
            Self::Prague => "Prague",
            Self::Osaka => "Osaka",
        }
    }
}

#[derive(Clone, Debug)]
pub struct ExecuteRequest {
    pub bytecode: Vec<u8>,
    pub calldata: Vec<u8>,
    pub gas_limit: u64,
    pub fork: Fork,
}

impl Default for ExecuteRequest {
    fn default() -> Self {
        Self {
            bytecode: vec![0],
            calldata: Vec::new(),
            gas_limit: DEFAULT_GAS_LIMIT,
            fork: Fork::Osaka,
        }
    }
}

#[derive(Debug, thiserror::Error)]
pub enum ExecuteError {
    #[error("execution failed: {0}")]
    Evm(String),
    #[error("invalid hexadecimal input: {0}")]
    Hex(String),
    #[error("invalid replay witness: {0}")]
    Witness(String),
    #[error("replay witness is incomplete: missing {accounts_len} accounts and {storage_len} storage slots", accounts_len = .accounts.len(), storage_len = .storage.len())]
    IncompleteWitness {
        accounts: Vec<Address>,
        storage: Vec<(Address, B256)>,
    },
}

pub fn decode_hex(input: &str) -> Result<Vec<u8>, ExecuteError> {
    hex::decode(input.trim().trim_start_matches("0x"))
        .map_err(|error| ExecuteError::Hex(error.to_string()))
}

pub fn execute(request: ExecuteRequest) -> Result<WireResult, ExecuteError> {
    Ok(engine::execute(engine::Request {
        bytecode: request.bytecode,
        calldata: request.calldata,
        gas_limit: request.gas_limit,
        fork: request.fork,
        trace: false,
    }))
}

pub fn trace(request: ExecuteRequest) -> Result<WireResult, ExecuteError> {
    Ok(engine::execute(engine::Request {
        bytecode: request.bytecode,
        calldata: request.calldata,
        gas_limit: request.gas_limit,
        fork: request.fork,
        trace: true,
    }))
}

/// Deploys constructor bytecode and calls the resulting contract in one
/// isolated, in-memory chain. Constructor state is committed before the call.
pub fn deploy_and_call(
    initcode: Vec<u8>,
    calldata: Vec<u8>,
    gas_limit: u64,
    fork: Fork,
    include_trace: bool,
) -> Result<WireResult, ExecuteError> {
    engine::deploy_and_call(initcode, calldata, gas_limit, fork, include_trace)
        .map_err(|error| ExecuteError::Evm(error.into()))
}

pub fn deploy_bytecode(
    initcode: Vec<u8>,
    gas_limit: u64,
    fork: Fork,
    include_trace: bool,
) -> Result<WireResult, ExecuteError> {
    Ok(engine::deploy(initcode, gas_limit, fork, include_trace))
}

/// Executes a complete, self-contained Mainnet witness with the embedded engine.
/// This path never contacts RPC or delegates execution to another client.
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

    let mut world = state::WorldState::default();
    world.enable_missing_tracking();
    for (address, account) in &witness.prestate {
        world.mark_known_account(*address);
        for slot in account.storage.keys() {
            world.mark_known_storage(*address, U256::from_be_bytes(slot.0));
        }
        let target = world.account_mut(*address);
        target.balance = account.balance.unwrap_or_default();
        target.nonce = account.nonce;
        target.code = account.code.to_vec();
        for (slot, value) in &account.storage {
            target
                .storage
                .insert(U256::from_be_bytes(slot.0), U256::from_be_bytes(value.0));
        }
    }
    let mut block_hashes = BTreeMap::new();
    for (number, hash) in &witness.block_hashes {
        let number = number.parse::<u64>().map_err(|error| {
            ExecuteError::Witness(format!("invalid blockHashes key {number:?}: {error}"))
        })?;
        block_hashes.insert(number, *hash);
    }

    let fork = Fork::for_timestamp(header.timestamp);
    let gas_price = U256::from(envelope.effective_gas_price(header.base_fee_per_gas));
    let mut execution = engine::transact(
        &mut world,
        engine::Transaction {
            caller,
            to: envelope.to(),
            value: envelope.value(),
            data: envelope.input().to_vec(),
            gas_limit: envelope.gas_limit(),
            gas_price,
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
            authorization_list: Vec::new(),
            set_code: false,
            max_fee_per_blob_gas: envelope.max_fee_per_blob_gas().map(U256::from),
            fork,
            environment: engine::Environment {
                chain_id: witness.chain_id,
                block_number: header.number,
                timestamp: header.timestamp,
                coinbase: header.beneficiary,
                block_gas_limit: header.gas_limit,
                base_fee: U256::from(header.base_fee_per_gas.unwrap_or_default()),
                prevrandao: U256::from_be_bytes(header.mix_hash.0),
                blob_base_fee: U256::ZERO,
                block_hashes,
                blob_hashes: envelope
                    .blob_versioned_hashes()
                    .unwrap_or_default()
                    .to_vec(),
            },
            trace: include_trace,
        },
    )
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

pub fn disassemble(bytecode: &[u8]) -> Vec<String> {
    let mut pc = 0usize;
    let mut lines = Vec::new();
    while pc < bytecode.len() {
        let opcode = bytecode[pc];
        let name = opcode::name(opcode).unwrap_or("UNKNOWN");
        let width = if (0x60..=0x7f).contains(&opcode) {
            usize::from(opcode - 0x5f)
        } else {
            0
        };
        let end = (pc + 1 + width).min(bytecode.len());
        let argument = if width == 0 {
            String::new()
        } else {
            format!(" 0x{}", hex::encode(&bytecode[pc + 1..end]))
        };
        lines.push(format!("{pc:04x}: {name}{argument}"));
        pc = end;
    }
    lines
}

pub fn assemble(input: &str) -> Result<Vec<u8>, ExecuteError> {
    if !input.contains(char::is_whitespace) {
        return decode_hex(input);
    }
    let mut output = Vec::new();
    for token in input.split_whitespace() {
        let upper = token.to_ascii_uppercase();
        if let Some(opcode) = opcode::by_name(&upper) {
            output.push(opcode);
        } else {
            output.extend(decode_hex(token)?);
        }
    }
    Ok(output)
}

/// Builds a deterministic, bounded causal evidence view from an exact trace.
/// The limit only affects presentation; execution has already completed.
pub fn build_evidence(result: &WireResult, profile: &str, limit: usize) -> serde_json::Value {
    let steps = result.trace.as_deref().unwrap_or_default();
    let mut candidates: Vec<&TraceStep> = steps
        .iter()
        .filter(|step| evidence_selects(profile, step))
        .collect();
    let candidate_count = candidates.len();
    let truncated = limit > 0 && candidates.len() > limit;
    if truncated {
        candidates.sort_by_key(|step| (std::cmp::Reverse(evidence_priority(step)), step.index));
        candidates.truncate(limit);
        candidates.sort_by_key(|step| step.index);
    }
    let events: Vec<_> = candidates
        .iter()
        .map(|step| {
            json!({
                "step": step.index,
                "depth": step.depth,
                "address": step.address,
                "pc": step.pc,
                "op": step.opcode_name,
                "gas": {
                    "before": step.gas_before,
                    "after": step.gas_after,
                    "used": step.gas_before.saturating_sub(step.gas_after),
                    "staticCost": step.gas_before.saturating_sub(step.gas_after)
                },
                "stack": { "before": step.stack_before, "after": step.stack_after },
                "halt": matches!(step.opcode_name.as_str(), "STOP" | "RETURN" | "REVERT" | "INVALID" | "SELFDESTRUCT"),
                "reverted": step.opcode_name == "REVERT",
                "error": step.halt_class,
                "why": evidence_explanation(step)
            })
        })
        .collect();
    json!({
        "schema": echoevm_protocol::EVIDENCE_SCHEMA,
        "profile": profile,
        "execution": {
            "status": format!("{:?}", result.status).to_lowercase(),
            "gasUsed": result.gas_used,
            "returnData": result.return_data,
            "totalSteps": steps.len(),
            "error": result.error
        },
        "events": events,
        "links": [],
        "selection": {
            "candidates": candidate_count,
            "selected": candidates.len(),
            "omitted": candidate_count.saturating_sub(candidates.len()),
            "truncated": truncated
        }
    })
}

fn evidence_selects(profile: &str, step: &TraceStep) -> bool {
    let op = step.opcode_name.as_str();
    match profile {
        "full" => true,
        "revert" => {
            matches!(op, "REVERT" | "INVALID" | "RETURN" | "STOP") || step.halt_class.is_some()
        }
        "storage" => matches!(op, "SLOAD" | "SSTORE" | "TLOAD" | "TSTORE"),
        "call" => matches!(
            op,
            "CALL"
                | "CALLCODE"
                | "DELEGATECALL"
                | "STATICCALL"
                | "CREATE"
                | "CREATE2"
                | "RETURN"
                | "REVERT"
        ),
        "abi" => matches!(
            op,
            "CALLDATALOAD"
                | "CALLDATACOPY"
                | "CALLDATASIZE"
                | "MLOAD"
                | "MSTORE"
                | "MSTORE8"
                | "RETURN"
                | "REVERT"
        ),
        "gas" => {
            step.gas_before.saturating_sub(step.gas_after) >= 100
                || matches!(op, "SSTORE" | "CALL" | "CREATE" | "CREATE2")
        }
        "arithmetic" => matches!(
            op,
            "ADD"
                | "SUB"
                | "MUL"
                | "DIV"
                | "SDIV"
                | "MOD"
                | "SMOD"
                | "ADDMOD"
                | "MULMOD"
                | "EXP"
                | "SIGNEXTEND"
        ),
        _ => {
            matches!(
                op,
                "REVERT"
                    | "INVALID"
                    | "SLOAD"
                    | "SSTORE"
                    | "TLOAD"
                    | "TSTORE"
                    | "CALL"
                    | "DELEGATECALL"
                    | "STATICCALL"
                    | "CREATE"
                    | "CREATE2"
                    | "RETURN"
                    | "SELFDESTRUCT"
            ) || step.halt_class.is_some()
        }
    }
}

fn evidence_priority(step: &TraceStep) -> u8 {
    if step.halt_class.is_some() || step.opcode_name == "REVERT" {
        5
    } else if matches!(step.opcode_name.as_str(), "SSTORE" | "TSTORE") {
        4
    } else if matches!(
        step.opcode_name.as_str(),
        "CALL" | "DELEGATECALL" | "STATICCALL" | "CREATE" | "CREATE2"
    ) {
        3
    } else {
        1
    }
}

fn evidence_explanation(step: &TraceStep) -> String {
    match step.opcode_name.as_str() {
        "SLOAD" => "Reads persistent contract storage.".into(),
        "SSTORE" => "Writes persistent contract storage if the frame commits.".into(),
        "TLOAD" | "TSTORE" => "Accesses transaction-scoped transient storage.".into(),
        "CALL" | "CALLCODE" | "DELEGATECALL" | "STATICCALL" => {
            "Transfers execution into another call frame.".into()
        }
        "CREATE" | "CREATE2" => "Creates a contract from initialization code.".into(),
        "REVERT" => "Reverts this frame and rolls back its state changes.".into(),
        "RETURN" => "Returns data successfully from this frame.".into(),
        op => format!("Executes {op} at program counter {}.", step.pc),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Address, B256};
    use echoevm_protocol::WitnessAccount;

    #[test]
    fn adds_and_returns_word() {
        let code = decode_hex("60016002015f5260205ff3").unwrap();
        let result = execute(ExecuteRequest {
            bytecode: code,
            ..Default::default()
        })
        .unwrap();
        assert_eq!(result.status, ExecutionStatus::Success);
        assert_eq!(
            result.return_data,
            "0x0000000000000000000000000000000000000000000000000000000000000003"
        );
    }

    #[test]
    fn disassembly_preserves_push_data() {
        assert_eq!(
            disassemble(&[0x60, 0x2a, 0x00]),
            ["0000: PUSH1 0x2a", "0002: STOP"]
        );
    }

    #[test]
    fn trace_emits_every_opcode() {
        let result = trace(ExecuteRequest {
            bytecode: decode_hex("600160020100").unwrap(),
            ..Default::default()
        })
        .unwrap();
        let trace = result.trace.unwrap();
        assert_eq!(trace.len(), 4);
        assert_eq!(trace[2].opcode_name, "ADD");
        assert!(trace[2].gas_before >= trace[2].gas_after);
    }

    #[test]
    fn replays_signed_mainnet_transaction_from_self_contained_state() {
        let raw = decode_hex("02f86f0102843b9aca0085029e7822d68298f094d9e1459a7a482635700cbc20bbaf52d495ab9c9680841b55ba3ac080a0c199674fcb29f353693dd779c017823b954b3c69dffa3cd6b2a6ff7888798039a028ca912de909e7e6cdef9cdcaf24c54dd8c1032946dfa1d85c206b32a9064fe8").unwrap();
        let sender: Address = "0x001e2b7de757ba469a57bf6b23d982458a07efce"
            .parse()
            .unwrap();
        let recipient: Address = "0xd9e1459a7a482635700cbc20bbaf52d495ab9c96"
            .parse()
            .unwrap();
        let header = Header {
            number: 19_500_000,
            gas_limit: 30_000_000,
            timestamp: MAINNET_CANCUN_TIME,
            base_fee_per_gas: Some(1),
            withdrawals_root: Some(B256::ZERO),
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            parent_beacon_block_root: Some(B256::ZERO),
            ..Default::default()
        };
        let witness = ReplayWitness {
            schema: WITNESS_SCHEMA.into(),
            chain_id: 1,
            block_hash: header.hash_slow(),
            transaction_index: 0,
            header: serde_json::to_value(header).unwrap(),
            transaction: raw.into(),
            prestate: BTreeMap::from([
                (
                    sender,
                    WitnessAccount {
                        balance: Some(U256::from(10_000_000_000_000_000_000_u128)),
                        nonce: 2,
                        ..Default::default()
                    },
                ),
                (recipient, WitnessAccount::default()),
                (Address::ZERO, WitnessAccount::default()),
            ]),
            block_hashes: BTreeMap::new(),
            source: Some("unit-test".into()),
        };
        let result = replay_witness(&witness, true).unwrap();
        assert_eq!(result.transaction.from, sender.to_string());
        assert_eq!(
            result.transaction.status, "success",
            "execution={:?}",
            result.execution
        );
        assert!(result.execution.trace.is_some());
        assert_eq!(result.witness.sha256.len(), 64);
    }
}
