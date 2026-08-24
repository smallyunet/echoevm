use alloy_dyn_abi::JsonAbiExt;
use alloy_json_abi::JsonAbi;
use alloy_primitives::{Address, B256, Bytes, U256};
use anyhow::{Context, Result, bail};
use clap::{Args, ValueEnum};
use echoevm_core::{BlockEnv, Fork, state::WorldState};
use echoevm_protocol::{
    ExecutionStatus, TEST_WITNESS_SCHEMA, TestAccount, TestEnvironment, TestExecutionContext,
    TestExpectation, TestFork, TestSourceMetadata, TestWitness,
};
use serde_json::Value;
use serde_json::json;
use sha2::{Digest, Sha256};
use std::{collections::BTreeMap, fs, path::PathBuf};

use crate::explain::{ExplainOutputArgs, explain_test_witness, write_explanation};
use crate::solidity::abi::{coerce, select_function};

mod support;
use support::*;

const DEFAULT_CALLER: &str = "0x1000000000000000000000000000000000000001";
const DEFAULT_TARGET: &str = "0x2000000000000000000000000000000000000002";
const HEVM_CHEATCODE_ADDRESS: &str = "7109709ecfa91a80626ff3989d68f67f5b1dd12d";

#[derive(Clone, Copy, Debug, ValueEnum)]
pub enum FoundryStatus {
    Success,
    Revert,
    Fault,
}

impl From<FoundryStatus> for ExecutionStatus {
    fn from(value: FoundryStatus) -> Self {
        match value {
            FoundryStatus::Success => Self::Success,
            FoundryStatus::Revert => Self::Revert,
            FoundryStatus::Fault => Self::Fault,
        }
    }
}

#[derive(Args, Debug)]
pub struct FoundryWitnessArgs {
    /// Foundry contract artifact, normally out/Source.sol/Contract.json.
    pub artifact: PathBuf,
    #[arg(long)]
    pub function: String,
    #[arg(long, default_value = "")]
    pub args: String,
    #[arg(long)]
    pub name: Option<String>,
    #[arg(long, default_value_t = 1_000_000)]
    pub gas: u64,
    #[arg(long, default_value = DEFAULT_CALLER)]
    pub caller: Address,
    #[arg(long, default_value = DEFAULT_TARGET)]
    pub target: Address,
    /// Call value as decimal or 0x-prefixed integer.
    #[arg(long, default_value = "0")]
    pub value: String,
    /// Caller balance as decimal or 0x-prefixed integer.
    #[arg(long, default_value = "0xffffffffffffffffffffffffffffffff")]
    pub caller_balance: String,
    /// Target storage entry SLOT=VALUE; repeat for every slot the call may read.
    #[arg(long = "storage")]
    pub storage: Vec<String>,
    #[arg(long, default_value_t = 1)]
    pub chain_id: u64,
    #[arg(long, default_value_t = 0)]
    pub block_number: u64,
    #[arg(long, default_value_t = 0)]
    pub timestamp: u64,
    #[arg(long, default_value = "0x0000000000000000000000000000000000000000")]
    pub coinbase: Address,
    #[arg(long, default_value_t = 30_000_000)]
    pub block_gas_limit: u64,
    /// Base fee as decimal or 0x-prefixed integer.
    #[arg(long, default_value = "0")]
    pub base_fee: String,
    /// PREVRANDAO as decimal or 0x-prefixed integer.
    #[arg(long, default_value = "0")]
    pub prevrandao: String,
    #[arg(long, value_enum)]
    pub expect_status: Option<FoundryStatus>,
    #[arg(long)]
    pub expect_return: Option<String>,
    #[arg(long, value_enum, default_value_t = FoundryFork::Osaka)]
    pub fork: FoundryFork,
    #[arg(long)]
    pub out: Option<PathBuf>,
}

#[derive(Clone, Copy, Debug, Default, ValueEnum)]
pub enum FoundryFork {
    Cancun,
    Prague,
    #[default]
    Osaka,
}

impl From<FoundryFork> for Fork {
    fn from(value: FoundryFork) -> Self {
        match value {
            FoundryFork::Cancun => Self::Cancun,
            FoundryFork::Prague => Self::Prague,
            FoundryFork::Osaka => Self::Osaka,
        }
    }
}

#[derive(Args, Debug)]
pub struct FoundryExplainArgs {
    /// Foundry contract artifact containing linked creation and runtime bytecode.
    pub artifact: PathBuf,
    #[arg(long)]
    pub test: String,
    #[arg(long, default_value = "")]
    pub args: String,
    #[arg(long, default_value = "")]
    pub constructor_args: String,
    #[arg(long, default_value_t = 3_000_000)]
    pub gas: u64,
    #[arg(long, default_value = DEFAULT_CALLER)]
    pub caller: Address,
    #[arg(long, default_value = "0xffffffffffffffffffffffffffffffff")]
    pub caller_balance: String,
    #[arg(long, default_value = "0")]
    pub value: String,
    #[arg(long, default_value_t = 1)]
    pub chain_id: u64,
    #[arg(long, default_value_t = 0)]
    pub block_number: u64,
    #[arg(long, default_value_t = 0)]
    pub timestamp: u64,
    #[arg(long, default_value = "0x0000000000000000000000000000000000000000")]
    pub coinbase: Address,
    #[arg(long, default_value_t = 30_000_000)]
    pub block_gas_limit: u64,
    #[arg(long, default_value = "0")]
    pub base_fee: String,
    #[arg(long, default_value = "0")]
    pub prevrandao: String,
    #[arg(long, value_enum, default_value_t = FoundryFork::Osaka)]
    pub fork: FoundryFork,
    /// Persist the exact self-contained witness used for the explanation.
    #[arg(long)]
    pub witness_out: Option<PathBuf>,
    #[command(flatten)]
    pub output: ExplainOutputArgs,
}

pub fn explain(args: FoundryExplainArgs) -> Result<()> {
    let artifact_bytes =
        fs::read(&args.artifact).with_context(|| format!("read {}", args.artifact.display()))?;
    let artifact: Value = serde_json::from_slice(&artifact_bytes)?;
    let abi: JsonAbi = serde_json::from_value(
        artifact
            .get("abi")
            .cloned()
            .context("Foundry artifact is missing abi")?,
    )?;
    let runtime = artifact_bytecode(&artifact, "/deployedBytecode/object", "deployedBytecode")?;
    let mut initcode = artifact_bytecode(&artifact, "/bytecode/object", "bytecode")?;
    if contains_cheatcode(&runtime) || contains_cheatcode(&initcode) {
        bail!("unsupported-capability: foundry-cheatcodes")
    }
    if let Some(constructor) = abi.constructor() {
        initcode.extend(
            constructor.abi_encode_input(&coerce(&constructor.inputs, &args.constructor_args)?)?,
        );
    } else if !args.constructor_args.trim().is_empty() {
        bail!("artifact has no constructor arguments")
    }
    let test = select_function(&abi, &args.test)?;
    let test_calldata = test.abi_encode_input(&coerce(&test.inputs, &args.args)?)?;
    let setup_calldata = abi
        .functions()
        .find(|function| function.name == "setUp")
        .map(|function| {
            if !function.inputs.is_empty() {
                bail!("unsupported-capability: foundry-set-up-with-arguments")
            }
            Ok(function.abi_encode_input(&[])?)
        })
        .transpose()?;

    let caller_balance = parse_word("caller balance", &args.caller_balance)?;
    let value = parse_word("value", &args.value)?;
    let base_fee = parse_word("base fee", &args.base_fee)?;
    let prevrandao = parse_word("prevrandao", &args.prevrandao)?;
    if args.block_gas_limit < args.gas {
        bail!("--block-gas-limit must be at least --gas")
    }
    let fork: Fork = args.fork.into();
    let environment = BlockEnv {
        chain_id: args.chain_id,
        block_number: args.block_number,
        timestamp: args.timestamp,
        coinbase: args.coinbase,
        block_gas_limit: args.block_gas_limit,
        base_fee,
        prevrandao,
        blob_base_fee: U256::ZERO,
        block_hashes: BTreeMap::new(),
        blob_hashes: Vec::new(),
    };
    let mut world = WorldState::default();
    world.account_mut(args.caller).balance = caller_balance;
    let target = args.caller.create(0);
    let deployed = run_step(
        &mut world,
        PreparationStep {
            caller: args.caller,
            to: None,
            value: U256::ZERO,
            data: initcode,
            gas_limit: args.gas,
            fork,
            environment: environment.clone(),
        },
    )?;
    reject_executed_cheatcode(&deployed)?;
    if deployed.status != ExecutionStatus::Success {
        bail!(
            "Foundry artifact constructor ended with {:?}",
            deployed.status
        )
    }
    let deployed_runtime = world
        .accounts
        .get(&target)
        .map(|account| account.code.clone())
        .filter(|code| !code.is_empty())
        .context("constructor produced no runtime bytecode")?;
    if deployed_runtime != runtime {
        bail!("artifact runtime does not match constructor output")
    }
    let setup_executed = setup_calldata.is_some();
    if let Some(setup) = setup_calldata {
        let setup_result = run_step(
            &mut world,
            PreparationStep {
                caller: args.caller,
                to: Some(target),
                value: U256::ZERO,
                data: setup,
                gas_limit: args.gas,
                fork,
                environment: environment.clone(),
            },
        )?;
        reject_executed_cheatcode(&setup_result)?;
        if setup_result.status != ExecutionStatus::Success {
            bail!("setUp() ended with {:?}", setup_result.status)
        }
    }

    let expected_status = args
        .output
        .expect_status
        .map(protocol_status)
        .unwrap_or(ExecutionStatus::Success);
    let expected_return = args
        .output
        .expect_return
        .as_deref()
        .map(|value| hex::decode(value.trim_start_matches("0x")).map(Bytes::from))
        .transpose()
        .context("invalid --expect-return hexadecimal")?;
    let accounts = snapshot_accounts(&world);
    let (source_file, source_locations) = foundry_source_locations(&artifact, &runtime);
    let witness = complete_reads(TestWitness {
        schema: TEST_WITNESS_SCHEMA.into(),
        name: format!("foundry:{}", test.signature()),
        bytecode: deployed_runtime.into(),
        calldata: test_calldata.into(),
        gas_limit: args.gas,
        fork: protocol_fork(args.fork),
        expectation: TestExpectation {
            status: Some(expected_status),
            return_data: expected_return,
            storage: BTreeMap::new(),
        },
        requires: Vec::new(),
        source: Some(TestSourceMetadata {
            file: source_file.or_else(|| Some(args.artifact.display().to_string())),
            contract: args
                .artifact
                .file_stem()
                .and_then(|value| value.to_str())
                .map(str::to_owned),
            function: Some(test.signature()),
            test: Some(test.name.clone()),
            locations: source_locations,
        }),
        context: Some(TestExecutionContext {
            caller: args.caller,
            target,
            value,
            gas_price: base_fee,
            accounts,
            environment: TestEnvironment {
                chain_id: args.chain_id,
                block_number: args.block_number,
                timestamp: args.timestamp,
                coinbase: args.coinbase,
                block_gas_limit: args.block_gas_limit,
                base_fee,
                prevrandao,
                ..Default::default()
            },
        }),
    })?;
    let witness_bytes = serde_json::to_vec_pretty(&witness)?;
    if let Some(path) = &args.witness_out {
        fs::write(path, &witness_bytes).with_context(|| format!("write {}", path.display()))?;
    }
    let context = witness.context.as_ref().expect("prepared context");
    let storage_slots = context
        .accounts
        .values()
        .map(|account| account.storage.len())
        .sum::<usize>();
    let input = json!({
        "kind": "foundry-test",
        "artifact": args.artifact,
        "test": test.signature(),
        "setupExecuted": setup_executed,
        "materializedAccounts": context.accounts.len(),
        "materializedStorageSlots": storage_slots,
        "deploymentAddress": target,
        "witnessSchema": TEST_WITNESS_SCHEMA,
        "witnessSha256": hex::encode(Sha256::digest(&witness_bytes)),
        "witnessPath": args.witness_out,
        "runtime": {"name": "EchoEVM", "version": env!("CARGO_PKG_VERSION")}
    });
    let document = explain_test_witness(
        &witness,
        input,
        args.output.effective_profile(),
        args.output.limit,
    )?;
    write_explanation(&document, &args.output.format)
}

pub fn export(args: FoundryWitnessArgs) -> Result<()> {
    let artifact_bytes =
        fs::read(&args.artifact).with_context(|| format!("read {}", args.artifact.display()))?;
    let artifact: Value = serde_json::from_slice(&artifact_bytes)?;
    let abi: JsonAbi = serde_json::from_value(
        artifact
            .get("abi")
            .cloned()
            .context("Foundry artifact is missing abi")?,
    )?;
    let function = select_function(&abi, &args.function)?;
    let calldata = function.abi_encode_input(&coerce(&function.inputs, &args.args)?)?;
    let runtime = artifact
        .pointer("/deployedBytecode/object")
        .or_else(|| artifact.get("deployedBytecode"))
        .and_then(Value::as_str)
        .context("Foundry artifact is missing deployedBytecode.object")?;
    let runtime = hex::decode(runtime.trim_start_matches("0x"))
        .context("deployed bytecode is not fully linked hexadecimal")?;
    if runtime.is_empty() {
        bail!("Foundry artifact has empty deployed bytecode")
    }

    let mut requires = Vec::new();
    if abi.functions().any(|candidate| candidate.name == "setUp") {
        requires.push("foundry-set-up".to_owned());
    }
    if hex::encode(&runtime).contains(HEVM_CHEATCODE_ADDRESS) {
        requires.push("foundry-cheatcodes".to_owned());
    }
    let storage = parse_storage(&args.storage)?;
    let caller_balance = args
        .caller_balance
        .parse::<U256>()
        .with_context(|| format!("invalid caller balance {:?}", args.caller_balance))?;
    let value = parse_word("value", &args.value)?;
    let base_fee = parse_word("base fee", &args.base_fee)?;
    let prevrandao = parse_word("prevrandao", &args.prevrandao)?;
    if args.block_gas_limit < args.gas {
        bail!("--block-gas-limit must be at least --gas")
    }
    let expectation = TestExpectation {
        status: args.expect_status.map(Into::into),
        return_data: args
            .expect_return
            .as_deref()
            .map(|value| hex::decode(value.trim_start_matches("0x")).map(Bytes::from))
            .transpose()
            .context("invalid --expect-return hexadecimal")?,
        storage: BTreeMap::new(),
    };
    let contract = args
        .artifact
        .file_stem()
        .and_then(|value| value.to_str())
        .map(str::to_owned);
    let witness = TestWitness {
        schema: TEST_WITNESS_SCHEMA.into(),
        name: args
            .name
            .unwrap_or_else(|| format!("foundry:{}", function.signature())),
        bytecode: runtime.into(),
        calldata: calldata.into(),
        gas_limit: args.gas,
        fork: match args.fork {
            FoundryFork::Cancun => TestFork::Cancun,
            FoundryFork::Prague => TestFork::Prague,
            FoundryFork::Osaka => TestFork::Osaka,
        },
        expectation,
        requires,
        source: Some(TestSourceMetadata {
            file: Some(args.artifact.display().to_string()),
            contract,
            function: Some(function.signature()),
            test: function
                .name
                .starts_with("test")
                .then(|| function.name.clone()),
            locations: Vec::new(),
        }),
        context: Some(TestExecutionContext {
            caller: args.caller,
            target: args.target,
            value,
            gas_price: base_fee,
            accounts: BTreeMap::from([
                (
                    args.caller,
                    TestAccount {
                        balance: caller_balance,
                        ..Default::default()
                    },
                ),
                (
                    args.target,
                    TestAccount {
                        storage,
                        ..Default::default()
                    },
                ),
            ]),
            environment: TestEnvironment {
                chain_id: args.chain_id,
                block_number: args.block_number,
                timestamp: args.timestamp,
                coinbase: args.coinbase,
                block_gas_limit: args.block_gas_limit,
                base_fee,
                prevrandao,
                ..Default::default()
            },
        }),
    };
    // Validate supported exports, but preserve requires in output so unsupported
    // Foundry dependencies remain machine-readable and fail closed downstream.
    if witness.requires.is_empty() {
        witness.validate()?;
    }
    let output = serde_json::to_vec_pretty(&witness)?;
    if let Some(path) = args.out {
        fs::write(&path, output).with_context(|| format!("write {}", path.display()))?;
    } else {
        println!("{}", String::from_utf8(output)?);
    }
    Ok(())
}

fn parse_word(label: &str, value: &str) -> Result<U256> {
    value
        .parse::<U256>()
        .with_context(|| format!("invalid {label} {value:?}"))
}

fn parse_storage(values: &[String]) -> Result<BTreeMap<B256, B256>> {
    values
        .iter()
        .map(|entry| {
            let (slot, value) = entry
                .split_once('=')
                .context("--storage must be SLOT=VALUE")?;
            Ok((
                slot.parse::<B256>().context("invalid storage slot")?,
                value.parse::<B256>().context("invalid storage value")?,
            ))
        })
        .collect()
}
