use alloy_dyn_abi::JsonAbiExt;
use alloy_json_abi::JsonAbi;
use alloy_primitives::{Address, B256, Bytes, U256};
use anyhow::{Context, Result, bail};
use clap::{Args, ValueEnum};
use echoevm_protocol::{
    ExecutionStatus, TEST_WITNESS_SCHEMA, TestAccount, TestEnvironment, TestExecutionContext,
    TestExpectation, TestFork, TestSourceMetadata, TestWitness,
};
use serde_json::Value;
use std::{collections::BTreeMap, fs, path::PathBuf};

use crate::solidity::abi::{coerce, select_function};

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
