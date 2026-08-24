#![cfg(feature = "official-fixtures")]

use alloy_consensus::{
    Transaction as AlloyTransaction, TxEnvelope, transaction::SignerRecoverable,
};
use alloy_eips::{Decodable2718, Encodable2718};
use alloy_primitives::{Address, B256, U256, keccak256};
use echoevm_core::{
    Authorization, BlockEnv, Fork, Transaction,
    state::{Account, WorldState},
    transact,
};
use echoevm_protocol::ExecutionStatus;
use serde::Deserialize;
use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
};

#[derive(Clone, Copy)]
struct Gate {
    fork: &'static str,
    target: &'static str,
    expected_files: usize,
    expected_transactions: usize,
    expected_accepted: usize,
    expected_rejected: usize,
}

const GATES: [Gate; 3] = [
    Gate {
        fork: "Cancun",
        target: "for_cancun",
        expected_files: 2_337,
        expected_transactions: 11_554,
        expected_accepted: 10_968,
        expected_rejected: 586,
    },
    Gate {
        fork: "Prague",
        target: "for_prague",
        expected_files: 2_471,
        expected_transactions: 13_851,
        expected_accepted: 13_063,
        expected_rejected: 788,
    },
    Gate {
        fork: "Osaka",
        target: "for_osaka",
        expected_files: 2_408,
        expected_transactions: 14_516,
        expected_accepted: 13_708,
        expected_rejected: 808,
    },
];

#[derive(Deserialize)]
struct Unit {
    env: JsonEnv,
    pre: BTreeMap<String, JsonAccount>,
    transaction: JsonTransaction,
    post: BTreeMap<String, Vec<JsonPost>>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct JsonEnv {
    current_coinbase: String,
    current_gas_limit: String,
    current_number: String,
    current_timestamp: String,
    #[serde(default)]
    current_random: Option<String>,
    #[serde(default)]
    current_difficulty: Option<String>,
    #[serde(default)]
    current_base_fee: Option<String>,
    #[serde(default)]
    current_excess_blob_gas: Option<String>,
}

#[derive(Deserialize)]
struct JsonAccount {
    nonce: String,
    balance: String,
    code: String,
    #[serde(default)]
    storage: BTreeMap<String, String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct JsonTransaction {
    nonce: String,
    gas_limit: Vec<String>,
    #[serde(default)]
    gas_price: Option<String>,
    #[serde(default)]
    max_fee_per_gas: Option<String>,
    #[serde(default)]
    max_priority_fee_per_gas: Option<String>,
    #[serde(default)]
    to: String,
    value: Vec<String>,
    data: Vec<String>,
    sender: String,
    #[serde(default)]
    blob_versioned_hashes: Vec<String>,
    #[serde(default)]
    max_fee_per_blob_gas: Option<String>,
    #[serde(default)]
    access_lists: Vec<Vec<JsonAccessListItem>>,
    #[serde(default)]
    authorization_list: Option<Vec<JsonAuthorization>>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct JsonAuthorization {
    chain_id: String,
    address: String,
    nonce: String,
    #[serde(default)]
    y_parity: Option<String>,
    #[serde(default)]
    v: Option<String>,
    r: String,
    s: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct JsonAccessListItem {
    address: String,
    storage_keys: Vec<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct JsonPost {
    indexes: JsonIndexes,
    txbytes: String,
    hash: String,
    logs: String,
    #[serde(default)]
    receipt: Option<JsonReceipt>,
    #[serde(default)]
    state: BTreeMap<String, JsonAccount>,
    #[serde(default)]
    expect_exception: Option<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct JsonReceipt {
    cumulative_gas_used: String,
    status: bool,
}

#[derive(Deserialize)]
struct JsonIndexes {
    data: usize,
    gas: usize,
    value: usize,
}

#[test]
fn official_state_fixtures_by_declared_fork_zero_skip() {
    let mut root = std::env::var_os("ECHOEVM_OFFICIAL_FIXTURES")
        .map(PathBuf::from)
        .expect("ECHOEVM_OFFICIAL_FIXTURES is required when official-fixtures is enabled");
    if root.is_relative() {
        root = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("../..")
            .join(root);
    }
    let selected_fork = std::env::var("ECHOEVM_OFFICIAL_FORK").ok();
    for gate in GATES {
        if selected_fork
            .as_deref()
            .is_some_and(|selected| selected != gate.fork)
        {
            continue;
        }
        run_gate(&root, gate);
    }
}

fn run_gate(root: &Path, gate: Gate) {
    let mut files = Vec::new();
    collect_json(
        &root.join("state_tests").join(gate.target),
        gate.fork,
        &mut files,
    );
    files.sort();
    assert_eq!(
        files.len(),
        gate.expected_files,
        "{} official file inventory changed",
        gate.fork
    );
    let mut transactions = 0usize;
    let mut rejected = 0usize;
    for path in &files {
        let bytes =
            fs::read(path).unwrap_or_else(|error| panic!("read {}: {error}", path.display()));
        let suite: BTreeMap<String, Unit> = serde_json::from_slice(&bytes)
            .unwrap_or_else(|error| panic!("decode {}: {error}", path.display()));
        for (name, unit) in suite {
            if name.starts_with('_') {
                continue;
            }
            let Some(cases) = unit.post.get(gate.fork) else {
                continue;
            };
            for (index, case) in cases.iter().enumerate() {
                transactions += 1;
                let mut world = decode_state(&unit.pre);
                if let Err(error) = validate_txbytes(case, &unit.transaction) {
                    if let Some(expected) = case.expect_exception.as_deref() {
                        assert_exception(path, &name, index, expected, &error, &unit, case);
                        assert_rejected_commitments(path, &name, index, &world, case);
                        rejected += 1;
                        continue;
                    }
                    panic!(
                        "{}::{name}[{index}] invalid txbytes: {error}",
                        path.display()
                    );
                }
                let transaction = build_transaction(&unit, case, gate).unwrap_or_else(|error| {
                    panic!("{}::{name}[{index}] build tx: {error}", path.display())
                });
                match (
                    transact(&mut world, transaction),
                    case.expect_exception.as_deref(),
                ) {
                    (Err(error), Some(expected)) => {
                        assert_exception(path, &name, index, expected, error, &unit, case);
                        assert_rejected_commitments(path, &name, index, &world, case);
                        rejected += 1;
                    }
                    (Err(error), None) => panic!(
                        "{}::{name}[{index}] unexpectedly rejected: {error}",
                        path.display()
                    ),
                    (Ok(_), Some(expected)) => panic!(
                        "{}::{name}[{index}] expected {expected}, transaction was accepted",
                        path.display()
                    ),
                    (Ok(result), None) => {
                        assert_state(path, &name, index, &world, &case.state);
                        assert_commitments(path, &name, index, &world, &result, case);
                    }
                }
            }
        }
    }
    assert_eq!(
        transactions, gate.expected_transactions,
        "{} transaction inventory",
        gate.fork
    );
    assert_eq!(
        rejected, gate.expected_rejected,
        "{} rejected count",
        gate.fork
    );
    assert_eq!(
        transactions - rejected,
        gate.expected_accepted,
        "{} accepted count",
        gate.fork
    );
    println!(
        "OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files={} transactions={} accepted={} rejected={} fork={} skipped=0",
        files.len(),
        transactions,
        transactions - rejected,
        rejected,
        gate.fork
    );
}

fn assert_commitments(
    path: &Path,
    name: &str,
    index: usize,
    world: &WorldState,
    result: &echoevm_protocol::ExecutionResult,
    case: &JsonPost,
) {
    let expected_root = case.hash.parse::<B256>().expect("fixture state root");
    assert_eq!(
        world.state_root(),
        expected_root,
        "{}::{name}[{index}] state root",
        path.display()
    );
    let expected_logs = case.logs.parse::<B256>().expect("fixture logs hash");
    assert_eq!(
        keccak256(alloy_rlp::encode(&world.logs)),
        expected_logs,
        "{}::{name}[{index}] logs hash",
        path.display()
    );
    let receipt = case.receipt.as_ref().expect("accepted fixture receipt");
    assert_eq!(
        result.gas_used,
        quantity_u64(&receipt.cumulative_gas_used).expect("receipt gas"),
        "{}::{name}[{index}] receipt gas",
        path.display()
    );
    assert_eq!(
        result.status == ExecutionStatus::Success,
        receipt.status,
        "{}::{name}[{index}] receipt status",
        path.display()
    );
}

fn validate_txbytes(case: &JsonPost, transaction: &JsonTransaction) -> Result<(), String> {
    let bytes = decode_bytes(&case.txbytes)?;
    let mut raw = bytes.as_slice();
    let envelope = TxEnvelope::decode_2718(&mut raw)
        .map_err(|error| format!("InvalidSignatureVrs: decode envelope: {error}"))?;
    if !raw.is_empty() {
        return Err("InvalidSignatureVrs: trailing transaction bytes".into());
    }
    if envelope.encoded_2718() != bytes {
        return Err("InvalidSignatureVrs: non-canonical transaction encoding".into());
    }
    if envelope.chain_id().is_some_and(|chain_id| chain_id != 1) {
        return Err(format!(
            "InvalidChainId: transaction chain id {:?} != fixture chain id 1",
            envelope.chain_id()
        ));
    }
    let signer = envelope
        .recover_signer()
        .map_err(|error| format!("InvalidSignatureVrs: recover sender: {error}"))?;
    let expected = parse_address(&transaction.sender)?;
    if signer != expected {
        return Err(format!(
            "InvalidSignatureVrs: recovered sender {signer} != fixture sender {expected}"
        ));
    }
    Ok(())
}

fn assert_exception(
    path: &Path,
    name: &str,
    index: usize,
    expected: &str,
    actual: &str,
    unit: &Unit,
    case: &JsonPost,
) {
    let category = match actual.split(':').next().unwrap_or(actual) {
        "GasLimitExceedsBlockGasLimit" => "GAS_ALLOWANCE_EXCEEDED",
        "GasLimitExceedsMaximum" => "GAS_LIMIT_EXCEEDS_MAXIMUM",
        "CreateInitCodeSizeLimit" => "INITCODE_SIZE_EXCEEDED",
        "InsufficientFunds" => "INSUFFICIENT_ACCOUNT_FUNDS",
        "UpfrontCostOverflow" | "GasPaymentOverflow" | "BlobGasPaymentOverflow" => {
            "GASLIMIT_PRICE_PRODUCT_OVERFLOW"
        }
        "BlobGasPriceTooLow" => "INSUFFICIENT_MAX_FEE_PER_BLOB_GAS",
        "MaxFeePerGasBelowBaseFee" => "INSUFFICIENT_MAX_FEE_PER_GAS",
        "CalldataFloorGasTooLow" => "INTRINSIC_GAS_BELOW_FLOOR_GAS_COST",
        "IntrinsicGasTooLow" => "INTRINSIC_GAS_TOO_LOW",
        "InvalidChainId" => "INVALID_CHAINID",
        "InvalidSignatureVrs" => {
            if unit.transaction.max_fee_per_blob_gas.is_some() && unit.transaction.to.is_empty() {
                "TYPE_3_TX_CONTRACT_CREATION"
            } else if unit.transaction.authorization_list.is_some()
                && unit.transaction.to.is_empty()
            {
                "TYPE_4_TX_CONTRACT_CREATION"
            } else if let Some(gas_price) = unit.transaction.gas_price.as_deref() {
                let gas_price = quantity(gas_price).expect("fixture gas price");
                let gas_limit = quantity(&unit.transaction.gas_limit[case.indexes.gas])
                    .expect("fixture gas limit");
                if gas_price.checked_mul(gas_limit).is_none() {
                    "GASLIMIT_PRICE_PRODUCT_OVERFLOW"
                } else {
                    "INVALID_SIGNATURE_VRS"
                }
            } else {
                "INVALID_SIGNATURE_VRS"
            }
        }
        "NonceOverflow" => "NONCE_IS_MAX",
        "NonceMismatch" => {
            let sender = parse_address(&unit.transaction.sender).expect("fixture sender");
            let state_nonce = unit
                .pre
                .get(&sender.to_string().to_lowercase())
                .map(|account| quantity_u64(&account.nonce).expect("state nonce"))
                .unwrap_or_default();
            let tx_nonce = quantity_u64(&unit.transaction.nonce).expect("tx nonce");
            if tx_nonce > state_nonce {
                "NONCE_MISMATCH_TOO_HIGH"
            } else {
                "NONCE_MISMATCH_TOO_LOW"
            }
        }
        "PriorityFeeAboveMaxFee" => "PRIORITY_GREATER_THAN_MAX_FEE_PER_GAS",
        "SenderNotExternallyOwned" => "SENDER_NOT_EOA",
        "TooManyBlobs" => "TYPE_3_TX_BLOB_COUNT_EXCEEDED",
        "BlobGasAllowanceExceeded" => "TYPE_3_TX_MAX_BLOB_GAS_ALLOWANCE_EXCEEDED",
        "BlobTransactionContractCreation" => "TYPE_3_TX_CONTRACT_CREATION",
        "InvalidBlobVersionedHash" => "TYPE_3_TX_INVALID_BLOB_VERSIONED_HASH",
        "BlobTransactionMissingBlobHashes" => "TYPE_3_TX_ZERO_BLOBS",
        "EmptyAuthorizationList" => "TYPE_4_EMPTY_AUTHORIZATION_LIST",
        "SetCodeTransactionContractCreation" => "TYPE_4_TX_CONTRACT_CREATION",
        other => panic!(
            "{}::{name}[{index}] unmapped rejection {other}: {actual}",
            path.display()
        ),
    };
    let category_matches = expected
        .split('|')
        .any(|candidate| candidate.ends_with(category))
        || (actual == "CalldataFloorGasTooLow"
            && expected
                .split('|')
                .any(|candidate| candidate.ends_with("INTRINSIC_GAS_TOO_LOW")));
    assert!(
        category_matches,
        "{}::{name}[{index}] expected {expected}, got {actual} ({category})",
        path.display()
    );
}

fn assert_rejected_commitments(
    path: &Path,
    name: &str,
    index: usize,
    world: &WorldState,
    case: &JsonPost,
) {
    assert_eq!(
        world.state_root(),
        case.hash.parse::<B256>().expect("rejected state root"),
        "{}::{name}[{index}] rejected state root",
        path.display()
    );
    assert_eq!(
        keccak256(alloy_rlp::encode(&world.logs)),
        case.logs.parse::<B256>().expect("rejected logs hash"),
        "{}::{name}[{index}] rejected logs hash",
        path.display()
    );
}

fn build_transaction(unit: &Unit, case: &JsonPost, gate: Gate) -> Result<Transaction, String> {
    let data = decode_bytes(
        unit.transaction
            .data
            .get(case.indexes.data)
            .ok_or("data index")?,
    )?;
    let gas_limit = quantity_u64(
        unit.transaction
            .gas_limit
            .get(case.indexes.gas)
            .ok_or("gas index")?,
    )?;
    let value = quantity(
        unit.transaction
            .value
            .get(case.indexes.value)
            .ok_or("value index")?,
    )?;
    let base_fee = unit
        .env
        .current_base_fee
        .as_deref()
        .map(quantity)
        .transpose()?
        .unwrap_or_default();
    let gas_price = if let Some(price) = &unit.transaction.gas_price {
        quantity(price)?
    } else {
        let max_fee = quantity(
            unit.transaction
                .max_fee_per_gas
                .as_deref()
                .ok_or("missing maxFeePerGas")?,
        )?;
        let priority = unit
            .transaction
            .max_priority_fee_per_gas
            .as_deref()
            .map(quantity)
            .transpose()?
            .unwrap_or_default();
        max_fee.min(base_fee.saturating_add(priority))
    };
    let max_fee_per_gas = unit
        .transaction
        .max_fee_per_gas
        .as_deref()
        .or(unit.transaction.gas_price.as_deref())
        .map(quantity)
        .transpose()?
        .unwrap_or_default();
    let block_number = quantity_u64(&unit.env.current_number)?;
    let block_hashes = if block_number == 0 {
        BTreeMap::new()
    } else {
        BTreeMap::from([(block_number - 1, keccak256(b"0"))])
    };
    Ok(Transaction {
        caller: parse_address(&unit.transaction.sender)?,
        to: (!unit.transaction.to.is_empty() && unit.transaction.to != "0x")
            .then(|| parse_address(&unit.transaction.to))
            .transpose()?,
        value,
        data,
        gas_limit,
        gas_price,
        max_fee_per_gas,
        max_priority_fee_per_gas: unit
            .transaction
            .max_priority_fee_per_gas
            .as_deref()
            .map(quantity)
            .transpose()?,
        nonce: quantity_u64(&unit.transaction.nonce)?,
        access_list: unit
            .transaction
            .access_lists
            .get(case.indexes.data)
            .into_iter()
            .flatten()
            .map(|item| {
                Ok((
                    parse_address(&item.address)?,
                    item.storage_keys
                        .iter()
                        .map(|key| quantity(key))
                        .collect::<Result<_, _>>()?,
                ))
            })
            .collect::<Result<_, String>>()?,
        authorization_list: unit
            .transaction
            .authorization_list
            .as_deref()
            .unwrap_or_default()
            .iter()
            .map(|authorization| {
                Ok(Authorization {
                    chain_id: quantity(&authorization.chain_id)?,
                    delegate: parse_address(&authorization.address)?,
                    nonce: quantity_u64(&authorization.nonce)?,
                    y_parity: quantity_u64(
                        authorization
                            .y_parity
                            .as_deref()
                            .or(authorization.v.as_deref())
                            .ok_or("authorization y parity")?,
                    )? as u8,
                    r: quantity(&authorization.r)?,
                    s: quantity(&authorization.s)?,
                })
            })
            .collect::<Result<_, String>>()?,
        set_code: unit.transaction.authorization_list.is_some(),
        max_fee_per_blob_gas: unit
            .transaction
            .max_fee_per_blob_gas
            .as_deref()
            .map(quantity)
            .transpose()?,
        fork: match gate.fork {
            "Cancun" => Fork::Cancun,
            "Prague" => Fork::Prague,
            "Osaka" => Fork::Osaka,
            _ => unreachable!(),
        },
        environment: BlockEnv {
            chain_id: 1,
            block_number,
            timestamp: quantity_u64(&unit.env.current_timestamp)?,
            coinbase: parse_address(&unit.env.current_coinbase)?,
            block_gas_limit: quantity_u64(&unit.env.current_gas_limit)?,
            base_fee,
            prevrandao: unit
                .env
                .current_random
                .as_deref()
                .or(unit.env.current_difficulty.as_deref())
                .map(quantity)
                .transpose()?
                .unwrap_or_default(),
            blob_base_fee: fake_exponential(
                1,
                unit.env
                    .current_excess_blob_gas
                    .as_deref()
                    .map(quantity_u64)
                    .transpose()?
                    .unwrap_or_default(),
                3_338_477,
            ),
            block_hashes,
            blob_hashes: unit
                .transaction
                .blob_versioned_hashes
                .iter()
                .map(|value| value.parse::<B256>().map_err(|error| error.to_string()))
                .collect::<Result<_, _>>()?,
        },
        trace: false,
    })
}

fn decode_state(input: &BTreeMap<String, JsonAccount>) -> WorldState {
    let mut world = WorldState::default();
    for (address, account) in input {
        world.accounts.insert(
            parse_address(address).expect("fixture address"),
            decode_account(account),
        );
    }
    world
}

fn decode_account(input: &JsonAccount) -> Account {
    Account {
        nonce: quantity_u64(&input.nonce).expect("fixture nonce"),
        balance: quantity(&input.balance).expect("fixture balance"),
        code: decode_bytes(&input.code).expect("fixture code"),
        storage: input
            .storage
            .iter()
            .filter_map(|(slot, value)| {
                let value = quantity(value).expect("value");
                (!value.is_zero()).then(|| (quantity(slot).expect("slot"), value))
            })
            .collect(),
    }
}

fn assert_state(
    path: &Path,
    name: &str,
    index: usize,
    actual: &WorldState,
    expected: &BTreeMap<String, JsonAccount>,
) {
    let expected = decode_state(expected);
    let refund = actual.refund;
    let actual: BTreeMap<_, _> = actual
        .accounts
        .iter()
        .filter(|(_, account)| !account.is_empty())
        .map(|(address, account)| (*address, account.clone()))
        .collect();
    let expected: BTreeMap<_, _> = expected
        .accounts
        .into_iter()
        .filter(|(_, account)| !account.is_empty())
        .collect();
    if actual != expected {
        let addresses = actual
            .keys()
            .chain(expected.keys())
            .copied()
            .collect::<std::collections::BTreeSet<_>>();
        let mut mismatches = Vec::new();
        for address in addresses {
            let actual_account = actual.get(&address);
            let expected_account = expected.get(&address);
            if actual_account != expected_account {
                mismatches.push(format!(
                    "{address}: actual={actual_account:?} expected={expected_account:?}"
                ));
            }
        }
        panic!(
            "{}::{name}[{index}] state mismatches: {}; refund={refund}",
            path.display(),
            mismatches.join("; ")
        );
    }
}

fn collect_json(root: &Path, fork: &str, output: &mut Vec<PathBuf>) {
    for entry in
        fs::read_dir(root).unwrap_or_else(|error| panic!("read {}: {error}", root.display()))
    {
        let path = entry.expect("fixture entry").path();
        if path.is_dir() {
            collect_json(&path, fork, output);
        } else if path
            .extension()
            .is_some_and(|extension| extension == "json")
        {
            let bytes = fs::read(&path).expect("fixture inventory");
            let marker = format!("\"{fork}\"");
            if bytes
                .windows(marker.len())
                .any(|window| window == marker.as_bytes())
            {
                output.push(path);
            }
        }
    }
}

fn parse_address(input: &str) -> Result<Address, String> {
    input.parse().map_err(|error| format!("{error}"))
}
fn decode_bytes(input: &str) -> Result<Vec<u8>, String> {
    hex::decode(input.trim_start_matches("0x")).map_err(|error| error.to_string())
}
fn quantity(input: &str) -> Result<U256, String> {
    U256::from_str_radix(input.trim_start_matches("0x"), 16).map_err(|error| error.to_string())
}
fn quantity_u64(input: &str) -> Result<u64, String> {
    quantity(input)?
        .try_into()
        .map_err(|_| format!("quantity exceeds u64: {input}"))
}

fn fake_exponential(factor: u64, numerator: u64, denominator: u64) -> U256 {
    let mut output = U256::ZERO;
    let mut accumulator = U256::from(factor) * U256::from(denominator);
    let mut index = 1u64;
    while !accumulator.is_zero() {
        output += accumulator;
        accumulator = accumulator * U256::from(numerator) / U256::from(denominator * index);
        index += 1;
    }
    output / U256::from(denominator)
}
