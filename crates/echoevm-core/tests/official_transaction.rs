#![cfg(feature = "official-fixtures")]

use alloy_consensus::{Transaction, TxEnvelope, transaction::SignerRecoverable};
use alloy_eips::{Decodable2718, Encodable2718};
use alloy_primitives::Address;
use serde_json::Value;
use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
};

#[test]
fn official_transaction_validation_by_declared_fork() {
    let root = fixture_root().join("transaction_tests");
    for fork in ["Cancun", "Prague", "Osaka"] {
        let mut files = Vec::new();
        collect_json(
            &root.join(format!("for_{}", fork.to_lowercase())),
            &mut files,
        );
        files.sort();
        let mut valid = 0usize;
        let mut rejected = 0usize;
        for path in &files {
            let units: BTreeMap<String, Value> =
                serde_json::from_slice(&fs::read(path).unwrap()).unwrap();
            for (name, unit) in units {
                if name.starts_with('_') {
                    continue;
                }
                let Some(result) = unit.get("result").and_then(|value| value.get(fork)) else {
                    continue;
                };
                let txbytes = unit["txbytes"].as_str().expect("txbytes");
                let bytes = hex::decode(txbytes.trim_start_matches("0x")).unwrap();
                let mut input = bytes.as_slice();
                let decoded = TxEnvelope::decode_2718(&mut input);
                if let Some(expected) = result.get("exception").and_then(Value::as_str) {
                    validate_rejection(expected, decoded.as_ref().ok(), !input.is_empty())
                        .unwrap_or_else(|| {
                            panic!(
                                "{}::{name}: expected {expected}, rejection was not established",
                                path.display()
                            )
                        });
                    rejected += 1;
                    continue;
                }
                let envelope =
                    decoded.unwrap_or_else(|error| panic!("{}::{name}: {error}", path.display()));
                assert!(
                    input.is_empty(),
                    "{}::{name}: trailing bytes",
                    path.display()
                );
                assert_eq!(
                    envelope.encoded_2718(),
                    bytes,
                    "{}::{name}: canonical bytes",
                    path.display()
                );
                if let Some(chain_id) = envelope.chain_id() {
                    assert!(
                        chain_id > 0,
                        "{}::{name}: protected chain id",
                        path.display()
                    );
                }
                let sender = envelope.recover_signer().unwrap_or_else(|error| {
                    panic!("{}::{name}: recover sender: {error}", path.display())
                });
                if let Some(expected) = result.get("sender").and_then(Value::as_str) {
                    assert_eq!(
                        sender,
                        expected.parse::<Address>().unwrap(),
                        "{}::{name}: sender",
                        path.display()
                    );
                }
                valid += 1;
            }
        }
        let (expected_files, expected_cases) = if fork == "Cancun" { (1, 1) } else { (13, 56) };
        assert_eq!(
            files.len(),
            expected_files,
            "{fork} transaction file inventory"
        );
        assert_eq!(
            valid + rejected,
            expected_cases,
            "{fork} transaction case inventory"
        );
        println!(
            "OFFICIAL TRANSACTION SUMMARY release=tests@v20.0.1 files={} valid={} declared_rejected={} fork={} skipped=0",
            files.len(),
            valid,
            rejected,
            fork
        );
    }
}

fn validate_rejection(expected: &str, envelope: Option<&TxEnvelope>, trailing: bool) -> Option<()> {
    if envelope.is_none() || trailing {
        return Some(());
    }
    if expected.contains("NONCE_OVERFLOW") {
        return None;
    }
    let authorizations = envelope?.authorization_list()?;
    if expected.contains("EMPTY_AUTHORIZATION_LIST") {
        return authorizations.is_empty().then_some(());
    }
    if expected.contains("INVALID_AUTHORIZATION_FORMAT") {
        return authorizations
            .iter()
            .any(|authorization| authorization.signature().is_err())
            .then_some(());
    }
    if expected.contains("INVALID_AUTHORITY_SIGNATURE") {
        return authorizations
            .iter()
            .any(|authorization| authorization.recover_authority().is_err())
            .then_some(());
    }
    None
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

fn collect_json(root: &Path, output: &mut Vec<PathBuf>) {
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
