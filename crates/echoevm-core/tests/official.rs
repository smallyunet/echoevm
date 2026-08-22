use revm::{
    Context, DatabaseCommit, ExecuteEvm, MainBuilder, MainContext,
    context::CfgEnv,
    database::State,
    primitives::{AddressMap, hardfork::SpecId},
    statetest_types::{AccountInfo as ExpectedAccount, SpecName, TestSuite},
};
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
fn official_osaka_state_fixtures_zero_skip() {
    let Some(mut root) = std::env::var_os("ECHOEVM_OFFICIAL_FIXTURES").map(PathBuf::from) else {
        eprintln!("official fixtures not configured; use scripts/test-official-fixtures.sh");
        return;
    };
    if root.is_relative() {
        root = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("../..")
            .join(root);
    }
    let mut files = Vec::new();
    for authored in ["prague", "osaka"] {
        collect_json(
            &root.join("state_tests/for_osaka").join(authored),
            &mut files,
        );
    }
    files.sort();
    assert!(
        files.len() >= 180,
        "official corpus shrank: {} files",
        files.len()
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
            let Some(cases) = unit.post.get(&SpecName::Osaka) else {
                continue;
            };
            let expected_cases = expected
                .get(name)
                .and_then(|unit| unit.post.get("Osaka"))
                .unwrap_or_else(|| panic!("{name} has no Osaka expected states"));
            assert_eq!(cases.len(), expected_cases.len(), "{name} Osaka case count");
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
                cfg.set_spec_and_mainnet_gas_params(SpecId::OSAKA);
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
    assert!(
        transactions >= 3_000,
        "official corpus shrank: {transactions} transactions"
    );
    println!(
        "OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files={} transactions={} accepted={} rejected={} fork=Osaka skipped=0",
        files.len(),
        transactions,
        transactions - rejected,
        rejected
    );
}

fn collect_json(root: &Path, output: &mut Vec<PathBuf>) {
    for entry in
        fs::read_dir(root).unwrap_or_else(|error| panic!("read {}: {error}", root.display()))
    {
        let path = entry.expect("read fixture entry").path();
        if path.is_dir() {
            collect_json(&path, output);
        } else if path
            .extension()
            .is_some_and(|extension| extension == "json")
        {
            let bytes = fs::read(&path).expect("read fixture for fork inventory");
            if bytes
                .windows(b"\"Osaka\"".len())
                .any(|window| window == b"\"Osaka\"")
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
