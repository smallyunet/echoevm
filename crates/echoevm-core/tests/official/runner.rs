use super::*;

pub(super) fn run_gate(root: &Path, gate: Gate) {
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

pub(super) fn collect_json(root: &Path, fork: &str, output: &mut Vec<PathBuf>) {
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
