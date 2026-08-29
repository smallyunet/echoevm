#![cfg(feature = "official-fixtures")]

use alloy_consensus::{Block, TxEnvelope};
use alloy_eips::{Encodable2718, eip4895::Withdrawal};
use alloy_primitives::{Address, B256, Bytes, U256};
use alloy_rlp::Decodable;
use echoevm_core::{ExecuteError, execute_block_witness};
use echoevm_protocol::{
    BLOCK_WITNESS_SCHEMA, BlockWithdrawal, BlockWitness, TestFork, WitnessAccount,
};
use serde::Deserialize;
use std::{collections::BTreeMap, fs, path::PathBuf};

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct FixtureUnit {
    network: String,
    pre: BTreeMap<Address, FixtureAccount>,
    blocks: Vec<FixtureBlock>,
}

#[derive(Deserialize)]
struct FixtureAccount {
    nonce: String,
    balance: String,
    code: Bytes,
    #[serde(default)]
    storage: BTreeMap<String, String>,
}

#[derive(Deserialize)]
struct FixtureBlock {
    rlp: Bytes,
    #[serde(default, rename = "expectException")]
    expect_exception: Option<String>,
}

#[test]
fn official_single_block_transition_smoke_by_declared_fork() {
    let root = fixture_root();
    let relative = "constantinople/eip1052_extcodehash/extcodehash/extcodehash_of_empty.json";
    let mut executed = 0usize;
    for fork in ["Cancun", "Prague", "Osaka"] {
        let path = root
            .join("blockchain_tests")
            .join(format!("for_{}", fork.to_lowercase()))
            .join(relative);
        let units: BTreeMap<String, FixtureUnit> =
            serde_json::from_slice(&fs::read(&path).unwrap()).unwrap();
        let unit = units
            .into_values()
            .find(|unit| unit.network == fork)
            .expect("declared fork fixture");
        assert_eq!(
            unit.blocks.len(),
            1,
            "{} must stay a single-block vector",
            path.display()
        );
        let block = decode_block(&unit.blocks[0].rlp);
        let mut witness = block_witness(block, decode_prestate(unit.pre), fork);
        let result = execute_fixture_block(&mut witness)
            .unwrap_or_else(|error| panic!("{}: {error}", path.display()));
        assert_eq!(result.fork, fork);
        executed += 1;
    }
    assert_eq!(executed, 3);
    println!("OFFICIAL BLOCK SUMMARY release=tests@v20.0.1 blocks={executed} forks=3 skipped=0");
}

#[test]
fn official_single_block_transition_corpus() {
    let root = fixture_root().join("blockchain_tests");
    let selected = std::env::var("ECHOEVM_OFFICIAL_FORK").ok();
    for fork in ["Cancun", "Prague", "Osaka"] {
        if selected.as_deref().is_some_and(|value| value != fork) {
            continue;
        }
        let mut files = Vec::new();
        collect_json(
            &root.join(format!("for_{}", fork.to_lowercase())),
            &mut files,
        );
        files.sort();
        let mut blocks = 0usize;
        let mut declared_rejected = 0usize;
        for path in &files {
            let units: BTreeMap<String, FixtureUnit> =
                serde_json::from_slice(&fs::read(path).unwrap()).unwrap();
            for (name, unit) in units {
                if unit.network != fork || unit.blocks.len() != 1 {
                    continue;
                }
                if unit.blocks[0].expect_exception.is_some() {
                    declared_rejected += 1;
                    continue;
                }
                let block = decode_block(&unit.blocks[0].rlp);
                let mut witness = block_witness(block, decode_prestate(unit.pre), fork);
                witness.source = Some(format!("{}::{name}", path.display()));
                execute_fixture_block(&mut witness)
                    .unwrap_or_else(|error| panic!("{}::{name}: {error}", path.display()));
                blocks += 1;
            }
        }
        let (expected_files, expected_blocks, expected_rejected) = match fork {
            "Cancun" => (2_401, 11_930, 748),
            "Prague" => (2_573, 14_621, 1_286),
            "Osaka" => (2_514, 15_371, 1_286),
            _ => unreachable!(),
        };
        assert_eq!(files.len(), expected_files, "{fork} block file inventory");
        assert_eq!(
            blocks, expected_blocks,
            "{fork} accepted single-block inventory"
        );
        assert_eq!(
            declared_rejected, expected_rejected,
            "{fork} rejected inventory"
        );
        println!(
            "OFFICIAL BLOCK CORPUS SUMMARY release=tests@v20.0.1 files={} accepted_single_blocks={} declared_rejected={} fork={} skipped=0",
            files.len(),
            blocks,
            declared_rejected,
            fork
        );
    }
}

fn fixture_root() -> PathBuf {
    let root = std::env::var_os("ECHOEVM_OFFICIAL_FIXTURES")
        .map(PathBuf::from)
        .expect("ECHOEVM_OFFICIAL_FIXTURES is required");
    if root.is_absolute() {
        root
    } else {
        PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("../..")
            .join(root)
    }
}

fn decode_block(bytes: &Bytes) -> Block<TxEnvelope> {
    let mut input = bytes.as_ref();
    let block = Block::<TxEnvelope>::decode(&mut input).expect("block RLP");
    assert!(input.is_empty());
    block
}

fn block_witness(
    block: Block<TxEnvelope>,
    prestate: BTreeMap<Address, WitnessAccount>,
    fork: &str,
) -> BlockWitness {
    let block_hash = block.header.hash_slow();
    let transactions = block
        .body
        .transactions
        .iter()
        .map(|transaction| transaction.encoded_2718().into())
        .collect();
    let withdrawals = block
        .body
        .withdrawals
        .unwrap_or_default()
        .into_iter()
        .map(
            |Withdrawal {
                 index,
                 validator_index,
                 address,
                 amount,
             }| BlockWithdrawal {
                index,
                validator_index,
                address,
                amount,
            },
        )
        .collect();
    let block_hashes = (block.header.number > 0)
        .then(|| {
            (
                (block.header.number - 1).to_string(),
                block.header.parent_hash,
            )
        })
        .into_iter()
        .collect();
    BlockWitness {
        schema: BLOCK_WITNESS_SCHEMA.into(),
        chain_id: 1,
        fork: match fork {
            "Cancun" => TestFork::Cancun,
            "Prague" => TestFork::Prague,
            "Osaka" => TestFork::Osaka,
            _ => unreachable!(),
        },
        block_hash,
        header: serde_json::to_value(block.header).unwrap(),
        transactions,
        withdrawals,
        prestate,
        block_hashes,
        source: Some("tests@v20.0.1 blockchain fixture".into()),
    }
}

fn decode_prestate(input: BTreeMap<Address, FixtureAccount>) -> BTreeMap<Address, WitnessAccount> {
    input
        .into_iter()
        .map(|(address, account)| {
            (
                address,
                WitnessAccount {
                    exists: Some(true),
                    balance: Some(quantity(&account.balance)),
                    nonce: quantity(&account.nonce).to::<u64>(),
                    code: account.code,
                    storage: account
                        .storage
                        .into_iter()
                        .map(|(slot, value)| {
                            (
                                B256::from(quantity(&slot).to_be_bytes::<32>()),
                                B256::from(quantity(&value).to_be_bytes::<32>()),
                            )
                        })
                        .collect(),
                    storage_complete: false,
                },
            )
        })
        .collect()
}

fn quantity(value: &str) -> U256 {
    U256::from_str_radix(value.trim_start_matches("0x"), 16).unwrap()
}

fn execute_fixture_block(
    witness: &mut BlockWitness,
) -> Result<echoevm_protocol::BlockExecutionResult, ExecuteError> {
    // Blockchain fixtures encode only allocated prestate. Materialize every
    // discovered omission as the fixture-defined empty account/zero slot so the
    // production executor can retain its explicit fail-closed witness contract.
    for _ in 0..32 {
        match execute_block_witness(witness, None) {
            Ok(result) => return Ok(result),
            Err(ExecuteError::IncompleteWitness { accounts, storage }) => {
                for address in accounts {
                    witness.prestate.entry(address).or_insert(WitnessAccount {
                        exists: Some(false),
                        balance: None,
                        nonce: 0,
                        code: Bytes::new(),
                        storage: BTreeMap::new(),
                        storage_complete: true,
                    });
                }
                for (address, slot) in storage {
                    let account = witness.prestate.entry(address).or_insert(WitnessAccount {
                        exists: Some(false),
                        balance: None,
                        nonce: 0,
                        code: Bytes::new(),
                        storage: BTreeMap::new(),
                        storage_complete: true,
                    });
                    if account.exists != Some(false) {
                        account.storage.insert(slot, B256::ZERO);
                    }
                }
            }
            Err(error) => return Err(error),
        }
    }
    Err(ExecuteError::Witness(
        "fixture missing-read discovery did not converge".into(),
    ))
}

fn collect_json(root: &std::path::Path, output: &mut Vec<PathBuf>) {
    for entry in fs::read_dir(root).unwrap() {
        let path = entry.unwrap().path();
        if path.is_dir() {
            collect_json(&path, output);
        } else if path
            .extension()
            .is_some_and(|extension| extension == "json")
        {
            output.push(path);
        }
    }
}
