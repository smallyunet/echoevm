use echoevm_core::{ExecuteRequest, Fork, decode_hex, opcode, trace};
use echoevm_protocol::ExecutionStatus;
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::collections::BTreeSet;

const VECTORS: &str = include_str!("../../../tests/conformance/bytecode-vectors.json");

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct Matrix {
    schema: String,
    engine_dependency: EngineDependency,
    declared_forks: Vec<String>,
    vectors: Vec<Vector>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct EngineDependency {
    name: String,
    version: String,
    registered_opcode_bytes: usize,
    registered_opcode_sha256: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct Vector {
    name: String,
    category: String,
    fork: String,
    bytecode: String,
    status: String,
    return_data: String,
    gas_used: u64,
    error: Option<String>,
    halt_class: String,
    trace_opcodes: Vec<String>,
}

#[test]
fn registered_opcode_inventory_is_stable() {
    let matrix: Matrix = serde_json::from_str(VECTORS).expect("decode bytecode matrix");
    assert_eq!(matrix.schema, "echoevm.bytecode-conformance.v1");
    assert_eq!(matrix.engine_dependency.name, "EchoEVM");
    assert_eq!(matrix.engine_dependency.version, env!("CARGO_PKG_VERSION"));

    let registered = opcode::inventory();
    assert_eq!(
        registered.len(),
        matrix.engine_dependency.registered_opcode_bytes,
        "registered opcode inventory changed; review fork activation and vectors before updating the matrix"
    );
    let inventory = registered
        .iter()
        .map(|(byte, name)| format!("{byte:02x}:{name}\n"))
        .collect::<String>();
    let inventory_hash = hex::encode(Sha256::digest(inventory.as_bytes()));
    assert_eq!(
        inventory_hash, matrix.engine_dependency.registered_opcode_sha256,
        "registered opcode bytes or names changed; review activation and vectors"
    );
    for (byte, name) in [
        (0x00, "STOP"),
        (0x1e, "CLZ"),
        (0x49, "BLOBHASH"),
        (0x5c, "TLOAD"),
        (0x5d, "TSTORE"),
        (0x5e, "MCOPY"),
        (0x5f, "PUSH0"),
        (0xe6, "DUPN"),
        (0xf5, "CREATE2"),
        (0xfd, "REVERT"),
    ] {
        assert_eq!(opcode::name(byte), Some(name));
    }
}

#[test]
fn independent_bytecode_vectors_match_exact_results_and_traces() {
    let matrix: Matrix = serde_json::from_str(VECTORS).expect("decode bytecode matrix");
    assert_eq!(matrix.declared_forks, ["Cancun", "Prague", "Osaka"]);
    assert_eq!(matrix.vectors.len(), 15, "vector count must not shrink");

    let categories: BTreeSet<_> = matrix
        .vectors
        .iter()
        .map(|vector| vector.category.as_str())
        .collect();
    let required: BTreeSet<_> = [
        "arithmetic",
        "control-flow",
        "eof-boundary",
        "fork-activation",
        "future-opcode-boundary",
        "invalid-opcode",
        "precompile",
        "returndata",
        "revert",
        "stack",
        "transient-storage",
    ]
    .into_iter()
    .collect();
    assert_eq!(
        categories, required,
        "required conformance category changed"
    );

    for vector in &matrix.vectors {
        let result = trace(ExecuteRequest {
            bytecode: decode_hex(&vector.bytecode).expect("valid vector bytecode"),
            calldata: Vec::new(),
            gas_limit: 15_000_000,
            fork: parse_fork(&vector.fork),
        })
        .unwrap_or_else(|error| panic!("{} execution error: {error}", vector.name));

        assert_eq!(
            status_name(&result.status),
            vector.status,
            "{} status",
            vector.name
        );
        assert_eq!(
            result.return_data, vector.return_data,
            "{} return data",
            vector.name
        );
        assert_eq!(result.gas_used, vector.gas_used, "{} gas", vector.name);
        assert_eq!(
            result.error.as_deref(),
            vector.error.as_deref(),
            "{} error",
            vector.name
        );

        let steps = result.trace.expect("trace requested");
        let opcodes: Vec<_> = steps.iter().map(|step| step.opcode_name.as_str()).collect();
        let expected: Vec<_> = vector.trace_opcodes.iter().map(String::as_str).collect();
        assert_eq!(opcodes, expected, "{} normalized opcode trace", vector.name);
        assert_eq!(
            steps.last().and_then(|step| step.halt_class.as_deref()),
            Some(vector.halt_class.as_str()),
            "{} halt class",
            vector.name
        );
    }

    println!(
        "BYTECODE CONFORMANCE SUMMARY vectors={} categories={} forks={} registered_opcodes={} skipped=0",
        matrix.vectors.len(),
        categories.len(),
        matrix.declared_forks.len(),
        matrix.engine_dependency.registered_opcode_bytes,
    );
}

fn parse_fork(value: &str) -> Fork {
    match value {
        "Cancun" => Fork::Cancun,
        "Prague" => Fork::Prague,
        "Osaka" => Fork::Osaka,
        other => panic!("unsupported vector fork {other}"),
    }
}

fn status_name(status: &ExecutionStatus) -> &'static str {
    match status {
        ExecutionStatus::Success => "success",
        ExecutionStatus::Revert => "revert",
        ExecutionStatus::Fault => "fault",
    }
}
