#![cfg(feature = "official-fixtures")]

#[path = "official/assertions.rs"]
mod assertions;
#[path = "official/runner.rs"]
mod runner;
#[path = "official/transaction.rs"]
mod transaction;

use assertions::*;
use runner::*;
use transaction::*;

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
