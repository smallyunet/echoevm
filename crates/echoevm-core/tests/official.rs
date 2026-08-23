#![cfg(feature = "official-fixtures")]

use alloy_primitives::{Address, B256, U256};
use echoevm_core::{
    Authorization, BlockEnv, Fork, Transaction,
    state::{Account, WorldState},
    transact,
};
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
    authored: &'static [&'static str],
    expected_files: usize,
    expected_transactions: usize,
    expected_accepted: usize,
    expected_rejected: usize,
}

const GATES: [Gate; 3] = [
    Gate {
        fork: "Cancun",
        target: "for_cancun",
        authored: &["cancun"],
        expected_files: 63,
        expected_transactions: 1_456,
        expected_accepted: 1_303,
        expected_rejected: 153,
    },
    Gate {
        fork: "Prague",
        target: "for_prague",
        authored: &["prague"],
        expected_files: 134,
        expected_transactions: 2_195,
        expected_accepted: 1_998,
        expected_rejected: 197,
    },
    Gate {
        fork: "Osaka",
        target: "for_osaka",
        authored: &["prague", "osaka"],
        expected_files: 187,
        expected_transactions: 3_461,
        expected_accepted: 3_244,
        expected_rejected: 217,
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
    #[serde(default)]
    state: BTreeMap<String, JsonAccount>,
    #[serde(default)]
    expect_exception: Option<String>,
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
    for gate in GATES {
        run_gate(&root, gate);
    }
}

fn run_gate(root: &Path, gate: Gate) {
    let mut files = Vec::new();
    for authored in gate.authored {
        collect_json(
            &root.join("state_tests").join(gate.target).join(authored),
            gate.fork,
            &mut files,
        );
    }
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
                let transaction = build_transaction(&unit, case, gate).unwrap_or_else(|error| {
                    panic!("{}::{name}[{index}] build tx: {error}", path.display())
                });
                match (
                    transact(&mut world, transaction),
                    case.expect_exception.as_deref(),
                ) {
                    (Err(_), Some(_)) => rejected += 1,
                    (Err(error), None) => panic!(
                        "{}::{name}[{index}] unexpectedly rejected: {error}",
                        path.display()
                    ),
                    (Ok(_), Some(expected)) => panic!(
                        "{}::{name}[{index}] expected {expected}, transaction was accepted",
                        path.display()
                    ),
                    (Ok(_), None) => assert_state(path, &name, index, &world, &case.state),
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
            block_number: quantity_u64(&unit.env.current_number)?,
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
            block_hashes: BTreeMap::new(),
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
            .map(|(slot, value)| {
                (
                    quantity(slot).expect("slot"),
                    quantity(value).expect("value"),
                )
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
    assert_eq!(
        actual,
        expected,
        "{}::{name}[{index}] state",
        path.display()
    );
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
