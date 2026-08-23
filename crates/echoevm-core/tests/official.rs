#![cfg(feature = "official-fixtures")]

use revm::{
    Context, DatabaseCommit, ExecuteEvm, MainBuilder, MainContext,
    context::CfgEnv,
    database::State,
    primitives::{AddressMap, hardfork::SpecId},
    statetest_types::{AccountInfo as ExpectedAccount, SpecName, TestSuite},
};

#[derive(Clone, Copy)]
struct Gate {
    fork: &'static str,
    target: &'static str,
    authored: &'static [&'static str],
    spec_id: SpecId,
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
        spec_id: SpecId::CANCUN,
        expected_files: 63,
        expected_transactions: 1_456,
        expected_accepted: 1_303,
        expected_rejected: 153,
    },
    Gate {
        fork: "Prague",
        target: "for_prague",
        authored: &["prague"],
        spec_id: SpecId::PRAGUE,
        expected_files: 134,
        expected_transactions: 2_195,
        expected_accepted: 1_998,
        expected_rejected: 197,
    },
    Gate {
        fork: "Osaka",
        target: "for_osaka",
        authored: &["prague", "osaka"],
        spec_id: SpecId::OSAKA,
        expected_files: 187,
        expected_transactions: 3_461,
        expected_accepted: 3_244,
        expected_rejected: 217,
    },
];
use serde::Deserialize;
use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
};

#[derive(Deserialize)]
struct ExpectedUnit {
    post: BTreeMap<String, Vec<ExpectedPost>>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ExpectedPost {
    #[serde(default)]
    state: AddressMap<ExpectedAccount>,
    #[serde(default)]
    expect_exception: Option<String>,
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
    let spec_name = match gate.fork {
        "Cancun" => SpecName::Cancun,
        "Prague" => SpecName::Prague,
        "Osaka" => SpecName::Osaka,
        other => panic!("unsupported official fixture gate {other}"),
    };
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
        "{} official file inventory changed; review the pinned corpus before updating the gate",
        gate.fork
    );

    let mut transactions = 0usize;
    let mut rejected = 0usize;
    for path in &files {
        let bytes =
            fs::read(path).unwrap_or_else(|error| panic!("read {}: {error}", path.display()));
        let suite: TestSuite = serde_json::from_slice(&bytes)
            .unwrap_or_else(|error| panic!("decode {}: {error}", path.display()));
        let expected: BTreeMap<String, ExpectedUnit> = serde_json::from_slice(&bytes)
            .unwrap_or_else(|error| panic!("decode expected state {}: {error}", path.display()));
        for (name, unit) in &suite.0 {
            if name.starts_with('_') {
                continue;
            }
            let Some(cases) = unit.post.get(&spec_name) else {
                continue;
            };
            let expected_cases = expected
                .get(name)
                .and_then(|unit| unit.post.get(gate.fork))
                .unwrap_or_else(|| panic!("{name} has no {} expected states", gate.fork));
            assert_eq!(
                cases.len(),
                expected_cases.len(),
                "{name} {} case count",
                gate.fork
            );
            for (index, (case, expected_case)) in cases.iter().zip(expected_cases).enumerate() {
                transactions += 1;
                let expected_exception = case
                    .expect_exception
                    .as_ref()
                    .or(expected_case.expect_exception.as_ref());
                let tx = match case.tx_env(unit) {
                    Ok(tx) => tx,
                    Err(error) if expected_exception.is_some() => {
                        let _ = error;
                        rejected += 1;
                        continue;
                    }
                    Err(error) => panic!("{}::{name}[{index}] build tx: {error}", path.display()),
                };
                let mut cfg = CfgEnv::default();
                cfg.chain_id = unit
                    .env
                    .current_chain_id
                    .unwrap_or(revm::primitives::U256::from(1))
                    .try_into()
                    .unwrap_or(u64::MAX);
                cfg.set_spec_and_mainnet_gas_params(gate.spec_id);
                cfg.set_max_blobs_per_tx(6);
                let block = unit.block_env(&mut cfg);
                let mut state = State::builder().with_cached_prestate(unit.state()).build();
                let execution = {
                    let mut evm = Context::mainnet()
                        .with_cfg(cfg)
                        .with_block(block)
                        .with_db(&mut state)
                        .build_mainnet();
                    evm.transact(tx)
                };
                match (execution, expected_exception) {
                    (Err(_), Some(_)) => {
                        rejected += 1;
                    }
                    (Err(error), None) => {
                        panic!(
                            "{}::{name}[{index}] unexpectedly rejected: {error:?}",
                            path.display()
                        )
                    }
                    (Ok(_), Some(expected)) => panic!(
                        "{}::{name}[{index}] expected {expected}, transaction was accepted",
                        path.display()
                    ),
                    (Ok(result), None) => {
                        state.commit(result.state);
                        assert_state(path, name, index, &state, &expected_case.state);
                    }
                }
            }
        }
    }
    assert_eq!(
        transactions, gate.expected_transactions,
        "{} official transaction inventory changed; review the pinned corpus before updating the gate",
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
        gate.fork,
    );
}

fn collect_json(root: &Path, fork: &str, output: &mut Vec<PathBuf>) {
    for entry in
        fs::read_dir(root).unwrap_or_else(|error| panic!("read {}: {error}", root.display()))
    {
        let path = entry.expect("read fixture entry").path();
        if path.is_dir() {
            collect_json(&path, fork, output);
        } else if path
            .extension()
            .is_some_and(|extension| extension == "json")
        {
            let bytes = fs::read(&path).expect("read fixture for fork inventory");
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

fn assert_state(
    path: &Path,
    name: &str,
    index: usize,
    state: &State<revm::database_interface::EmptyDB>,
    expected: &AddressMap<ExpectedAccount>,
) {
    let actual_accounts: AddressMap<_> = state
        .cache
        .accounts
        .iter()
        .filter_map(|(address, account)| {
            account.account.as_ref().map(|account| (*address, account))
        })
        .collect();
    assert_eq!(
        actual_accounts.len(),
        expected.len(),
        "{}::{name}[{index}] account count",
        path.display()
    );
    for (address, want) in expected {
        let got = actual_accounts.get(address).unwrap_or_else(|| {
            panic!(
                "{}::{name}[{index}] missing account {address}",
                path.display()
            )
        });
        assert_eq!(
            got.info.balance, want.balance,
            "{name}[{index}] {address} balance"
        );
        assert_eq!(
            got.info.nonce, want.nonce,
            "{name}[{index}] {address} nonce"
        );
        let code = got
            .info
            .code
            .as_ref()
            .map(|code| code.original_bytes())
            .or_else(|| {
                state
                    .cache
                    .contracts
                    .get(&got.info.code_hash)
                    .map(|code| code.original_bytes())
            })
            .unwrap_or_default();
        assert_eq!(
            code.as_ref(),
            want.code.as_ref(),
            "{name}[{index}] {address} code"
        );
        let storage: BTreeMap<_, _> = got
            .storage
            .iter()
            .filter_map(|(slot, value)| (!value.is_zero()).then_some((*slot, *value)))
            .collect();
        let want_storage: BTreeMap<_, _> = want
            .storage
            .iter()
            .filter_map(|(slot, value)| (!value.is_zero()).then_some((*slot, *value)))
            .collect();
        assert_eq!(storage, want_storage, "{name}[{index}] {address} storage");
    }
}
