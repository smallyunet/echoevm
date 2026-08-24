use super::{FoundryFork, HEVM_CHEATCODE_ADDRESS};
use crate::{explain::ExpectedStatus, solidity::source_map::instruction_pcs};
use alloy_primitives::{Address, B256, U256};
use anyhow::{Context, Result, bail};
use echoevm_core::{
    BlockEnv, ExecuteError, Fork, Transaction, execute_test_witness, state::WorldState, transact,
};
use echoevm_protocol::{ExecutionStatus, TestAccount, TestFork, TestWitness};
use serde_json::Value;
use std::collections::BTreeMap;

pub(super) fn artifact_bytecode(artifact: &Value, pointer: &str, legacy: &str) -> Result<Vec<u8>> {
    let value = artifact
        .pointer(pointer)
        .or_else(|| artifact.get(legacy))
        .and_then(Value::as_str)
        .with_context(|| format!("Foundry artifact is missing {pointer}"))?;
    let bytes = hex::decode(value.trim_start_matches("0x"))
        .with_context(|| format!("{legacy} is not fully linked hexadecimal"))?;
    if bytes.is_empty() {
        bail!("Foundry artifact has empty {legacy}")
    }
    Ok(bytes)
}

pub(super) fn foundry_source_locations(
    artifact: &Value,
    runtime: &[u8],
) -> (Option<String>, Vec<echoevm_protocol::TestSourceLocation>) {
    let source_id = artifact.get("id").and_then(Value::as_i64);
    let source_file = compilation_target(artifact);
    let source_map = artifact
        .pointer("/deployedBytecode/sourceMap")
        .and_then(Value::as_str);
    let (Some(source_id), Some(file), Some(source_map)) =
        (source_id, source_file.clone(), source_map)
    else {
        return (source_file, Vec::new());
    };
    let pcs = instruction_pcs(runtime);
    let mut start = -1_i64;
    let mut length = -1_i64;
    let mut file_id = -1_i64;
    let mut locations = Vec::new();
    for (index, segment) in source_map.split(';').enumerate() {
        if index >= pcs.len() {
            break;
        }
        let fields: Vec<_> = segment.split(':').collect();
        if fields.first().is_some_and(|value| !value.is_empty()) {
            start = fields[0].parse().unwrap_or(-1);
        }
        if fields.get(1).is_some_and(|value| !value.is_empty()) {
            length = fields[1].parse().unwrap_or(-1);
        }
        if fields.get(2).is_some_and(|value| !value.is_empty()) {
            file_id = fields[2].parse().unwrap_or(-1);
        }
        if file_id == source_id && start >= 0 && length >= 0 {
            locations.push(echoevm_protocol::TestSourceLocation {
                pc: pcs[index] as u64,
                file: file.clone(),
                start: start as usize,
                length: length as usize,
            });
        }
    }
    (Some(file), locations)
}

fn compilation_target(artifact: &Value) -> Option<String> {
    artifact
        .pointer("/metadata/settings/compilationTarget")
        .and_then(Value::as_object)
        .and_then(|targets| targets.keys().next().cloned())
        .or_else(|| {
            let raw: Value = serde_json::from_str(artifact.get("rawMetadata")?.as_str()?).ok()?;
            raw.pointer("/settings/compilationTarget")
                .and_then(Value::as_object)
                .and_then(|targets| targets.keys().next().cloned())
        })
}

pub(super) fn contains_cheatcode(code: &[u8]) -> bool {
    hex::encode(code).contains(HEVM_CHEATCODE_ADDRESS)
}

pub(super) fn protocol_fork(fork: FoundryFork) -> TestFork {
    match fork {
        FoundryFork::Cancun => TestFork::Cancun,
        FoundryFork::Prague => TestFork::Prague,
        FoundryFork::Osaka => TestFork::Osaka,
    }
}

pub(super) fn protocol_status(status: ExpectedStatus) -> ExecutionStatus {
    match status {
        ExpectedStatus::Success => ExecutionStatus::Success,
        ExpectedStatus::Revert => ExecutionStatus::Revert,
        ExpectedStatus::Fault => ExecutionStatus::Fault,
    }
}

pub(super) struct PreparationStep {
    pub caller: Address,
    pub to: Option<Address>,
    pub value: U256,
    pub data: Vec<u8>,
    pub gas_limit: u64,
    pub fork: Fork,
    pub environment: BlockEnv,
}

pub(super) fn run_step(
    world: &mut WorldState,
    step: PreparationStep,
) -> Result<echoevm_protocol::ExecutionResult> {
    let nonce = world
        .accounts
        .get(&step.caller)
        .map_or(0, |account| account.nonce);
    transact(
        world,
        Transaction {
            caller: step.caller,
            to: step.to,
            value: step.value,
            data: step.data,
            gas_limit: step.gas_limit,
            gas_price: step.environment.base_fee,
            max_fee_per_gas: step.environment.base_fee,
            max_priority_fee_per_gas: Some(U256::ZERO),
            nonce,
            access_list: Vec::new(),
            authorization_list: Vec::new(),
            set_code: false,
            max_fee_per_blob_gas: None,
            fork: step.fork,
            environment: step.environment,
            trace: true,
        },
    )
    .map_err(|error| anyhow::anyhow!("EchoEVM preparation step failed: {error}"))
}

pub(super) fn snapshot_accounts(world: &WorldState) -> BTreeMap<Address, TestAccount> {
    world
        .accounts
        .iter()
        .map(|(address, account)| {
            (
                *address,
                TestAccount {
                    balance: account.balance,
                    nonce: account.nonce,
                    code: account.code.clone().into(),
                    storage: account
                        .storage
                        .iter()
                        .map(|(slot, value)| {
                            (
                                B256::from(slot.to_be_bytes::<32>()),
                                B256::from(value.to_be_bytes::<32>()),
                            )
                        })
                        .collect(),
                },
            )
        })
        .collect()
}

pub(super) fn complete_reads(mut witness: TestWitness) -> Result<TestWitness> {
    for _ in 0..8 {
        match execute_test_witness(&witness, true) {
            Ok(execution) => {
                reject_executed_cheatcode(&execution)?;
                return Ok(witness);
            }
            Err(ExecuteError::IncompleteWitness { accounts, storage }) => {
                let context = witness.context.as_mut().expect("prepared context");
                for address in accounts {
                    context.accounts.entry(address).or_default();
                }
                for (address, slot) in storage {
                    context
                        .accounts
                        .entry(address)
                        .or_default()
                        .storage
                        .entry(slot)
                        .or_insert(B256::ZERO);
                }
            }
            Err(error) => return Err(error.into()),
        }
    }
    bail!("could not close the prepared witness read set after 8 passes")
}

pub(super) fn reject_executed_cheatcode(
    execution: &echoevm_protocol::ExecutionResult,
) -> Result<()> {
    let used = execution
        .trace
        .as_deref()
        .unwrap_or_default()
        .iter()
        .filter_map(|step| step.control.as_ref()?.target.as_deref())
        .any(|target| {
            target
                .trim_start_matches("0x")
                .eq_ignore_ascii_case(HEVM_CHEATCODE_ADDRESS)
        });
    if used {
        bail!("unsupported-capability: foundry-cheatcodes")
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use echoevm_protocol::{ControlFlow, TraceStep};

    #[test]
    fn rejects_dynamically_executed_hevm_target() {
        let execution = echoevm_protocol::ExecutionResult {
            trace: Some(vec![TraceStep {
                opcode_name: "CALL".into(),
                control: Some(ControlFlow {
                    kind: "call".into(),
                    target: Some(format!("0x{HEVM_CHEATCODE_ADDRESS}")),
                    destination: None,
                }),
                ..Default::default()
            }]),
            ..Default::default()
        };
        let error = reject_executed_cheatcode(&execution).unwrap_err();
        assert!(error.to_string().contains("foundry-cheatcodes"));
    }
}
