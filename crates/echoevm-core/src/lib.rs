//! Independent EchoEVM execution kernel.

mod behavior;
mod block;
mod bls;
mod bn254;
mod bytecode;
mod engine;
mod evidence;
mod explanation;
mod kzg;
pub mod opcode;
mod replay;
pub mod state;

pub use behavior::infer_behavior;
pub use block::{
    TransactionWitnessMaterialization, execute_block_witness, materialize_transaction_witness,
};
pub use bytecode::{assemble, disassemble};
pub use engine::{Authorization, Environment as BlockEnv, Transaction, transact};
pub use evidence::build_evidence;
pub use explanation::explain_evidence;
pub use replay::replay_witness;

use alloy_consensus::{
    Header, Transaction as AlloyTransaction, TxEnvelope, transaction::SignerRecoverable,
};
use alloy_eips::{Decodable2718, Typed2718};
use alloy_primitives::{Address, B256, U256};
use echoevm_protocol::{
    ExecutionResult as WireResult, ExecutionStatus, ReplayResult, ReplayWitness, TestFork,
    TestWitness, TraceStep, TransactionSummary, WITNESS_SCHEMA, WitnessProvenance,
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
    #[error("execution witness is incomplete: missing {accounts_len} accounts and {storage_len} storage slots", accounts_len = .accounts.len(), storage_len = .storage.len())]
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

/// Executes a strict test witness. Context-free v1 witnesses retain the
/// original isolated-call behavior; context-bearing witnesses use the same
/// transaction/state path as self-contained Mainnet replay.
pub fn execute_test_witness(
    witness: &TestWitness,
    include_trace: bool,
) -> Result<WireResult, ExecuteError> {
    witness
        .validate()
        .map_err(|error| ExecuteError::Witness(error.to_string()))?;
    let fork = match witness.fork {
        TestFork::Cancun => Fork::Cancun,
        TestFork::Prague => Fork::Prague,
        TestFork::Osaka => Fork::Osaka,
    };
    let Some(context) = &witness.context else {
        return Ok(engine::execute(engine::Request {
            bytecode: witness.bytecode.to_vec(),
            calldata: witness.calldata.to_vec(),
            gas_limit: witness.gas_limit,
            fork,
            trace: include_trace,
        }));
    };

    let mut world = state::WorldState::default();
    world.enable_missing_tracking();
    for (address, account) in &context.accounts {
        world.mark_known_account(*address);
        for slot in account.storage.keys() {
            world.mark_known_storage(*address, U256::from_be_bytes(slot.0));
        }
        let target = world.account_mut(*address);
        target.balance = account.balance;
        target.nonce = account.nonce;
        target.code = account.code.to_vec();
        target.storage = account
            .storage
            .iter()
            .map(|(slot, value)| (U256::from_be_bytes(slot.0), U256::from_be_bytes(value.0)))
            .collect();
    }
    world.mark_known_account(context.target);
    world.account_mut(context.target).code = witness.bytecode.to_vec();
    world.mark_known_account(context.environment.coinbase);

    let caller_nonce = context.accounts[&context.caller].nonce;
    let mut execution = engine::transact(
        &mut world,
        engine::Transaction {
            caller: context.caller,
            to: Some(context.target),
            value: context.value,
            data: witness.calldata.to_vec(),
            gas_limit: witness.gas_limit,
            gas_price: context.gas_price,
            max_fee_per_gas: context.gas_price,
            max_priority_fee_per_gas: Some(U256::ZERO),
            nonce: caller_nonce,
            access_list: Vec::new(),
            authorization_list: Vec::new(),
            set_code: false,
            max_fee_per_blob_gas: None,
            fork,
            environment: engine::Environment {
                chain_id: context.environment.chain_id,
                block_number: context.environment.block_number,
                timestamp: context.environment.timestamp,
                coinbase: context.environment.coinbase,
                block_gas_limit: context.environment.block_gas_limit,
                base_fee: context.environment.base_fee,
                prevrandao: context.environment.prevrandao,
                blob_base_fee: context.environment.blob_base_fee,
                block_hashes: context.environment.block_hashes.clone(),
                blob_hashes: Vec::new(),
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
    execution.storage = world
        .accounts
        .get(&context.target)
        .into_iter()
        .flat_map(|account| &account.storage)
        .map(|(slot, value)| (format!("0x{slot:064x}"), format!("0x{value:064x}")))
        .collect();
    Ok(execution)
}

/// Executes a complete, self-contained Mainnet witness with the embedded engine.
/// This path never contacts RPC or delegates execution to another client.
#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Address, B256};
    use echoevm_protocol::WitnessAccount;
    use echoevm_protocol::{
        TEST_WITNESS_SCHEMA, TestAccount, TestEnvironment, TestExecutionContext, TestExpectation,
        TestFork, TestWitness,
    };

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

    fn stateful_test_witness(storage: BTreeMap<B256, B256>) -> TestWitness {
        let caller = Address::from([0x10; 20]);
        let target = Address::from([0x20; 20]);
        TestWitness {
            schema: TEST_WITNESS_SCHEMA.into(),
            name: "stateful-sload".into(),
            bytecode: decode_hex("60005460005260206000f3").unwrap().into(),
            calldata: Default::default(),
            gas_limit: 100_000,
            fork: TestFork::Osaka,
            expectation: TestExpectation::default(),
            requires: Vec::new(),
            source: None,
            context: Some(TestExecutionContext {
                caller,
                target,
                value: U256::ZERO,
                gas_price: U256::ZERO,
                accounts: BTreeMap::from([
                    (
                        caller,
                        TestAccount {
                            balance: U256::from(1_000_000),
                            ..Default::default()
                        },
                    ),
                    (
                        target,
                        TestAccount {
                            storage,
                            ..Default::default()
                        },
                    ),
                ]),
                environment: TestEnvironment::default(),
            }),
        }
    }

    #[test]
    fn executes_stateful_test_witness_through_transaction_path() {
        let slot = B256::ZERO;
        let value = B256::from(U256::from(42).to_be_bytes::<32>());
        let result = execute_test_witness(
            &stateful_test_witness(BTreeMap::from([(slot, value)])),
            true,
        )
        .unwrap();
        assert_eq!(result.status, ExecutionStatus::Success);
        assert_eq!(result.return_data, format!("0x{:064x}", 42));
        assert!(result.trace.is_some());
    }

    #[test]
    fn stateful_test_witness_fails_closed_on_undeclared_slot() {
        let error =
            execute_test_witness(&stateful_test_witness(BTreeMap::new()), true).unwrap_err();
        assert!(matches!(error, ExecuteError::IncompleteWitness { .. }));
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
