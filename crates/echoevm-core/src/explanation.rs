use echoevm_protocol::{
    EXPLANATION_SCHEMA, ExplanationDocument, ExplanationEvidenceSummary, ExplanationExpectation,
    ExplanationFinding, ExplanationVerdict,
};
use serde_json::{Value, json};

mod findings;

pub fn explain_evidence(
    evidence: &Value,
    input: Value,
    expectation: ExplanationExpectation,
) -> ExplanationDocument {
    let execution = evidence
        .get("execution")
        .cloned()
        .unwrap_or_else(|| json!({}));
    let status = execution
        .get("status")
        .and_then(Value::as_str)
        .unwrap_or("unknown");
    let return_data = execution
        .get("returnData")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let status_mismatch = expectation
        .status
        .as_deref()
        .is_some_and(|expected| expected != status);
    let return_mismatch = expectation
        .return_data
        .as_deref()
        .is_some_and(|expected| !return_data_matches(expected, return_data));
    let storage_mismatch = !storage_matches(&expectation, &execution);
    let expectation_mismatch = status_mismatch || return_mismatch || storage_mismatch;

    let findings = findings::collect(evidence, status);

    let root_cause = choose_root_cause(
        &findings,
        status_mismatch,
        return_mismatch,
        storage_mismatch,
        status,
    );
    let verdict = verdict(
        status,
        status_mismatch,
        return_mismatch,
        storage_mismatch,
        root_cause.as_ref(),
    );
    let selection = evidence.get("selection").unwrap_or(&Value::Null);
    let summary = ExplanationEvidenceSummary {
        schema: evidence
            .get("schema")
            .and_then(Value::as_str)
            .unwrap_or("unknown")
            .into(),
        profile: evidence
            .get("profile")
            .and_then(Value::as_str)
            .unwrap_or("unknown")
            .into(),
        candidates: usize_field(selection, "candidates"),
        selected: usize_field(selection, "selected"),
        omitted: usize_field(selection, "omitted"),
        truncated: selection
            .get("truncated")
            .and_then(Value::as_bool)
            .unwrap_or(false),
    };
    let mut limitations = vec![
        "The explanation applies only to this execution input, fork, gas limit, and initial state."
            .into(),
        "Causal findings describe captured EVM execution facts, not a security audit or formal verification result."
            .into(),
    ];
    if summary.truncated {
        limitations.push(format!(
            "Bounded evidence omitted {} candidate events; rerun with a larger limit or full trace if needed.",
            summary.omitted
        ));
    }
    if root_cause.is_none() && expectation_mismatch {
        limitations.push(
            "The observed result differs from the expectation, but the selected evidence does not establish a specific causal mechanism."
                .into(),
        );
    }

    ExplanationDocument {
        schema: EXPLANATION_SCHEMA.into(),
        input,
        expectation,
        verdict,
        root_cause,
        findings,
        execution,
        evidence: summary,
        limitations,
    }
}

fn verdict(
    status: &str,
    status_mismatch: bool,
    return_mismatch: bool,
    storage_mismatch: bool,
    root_cause: Option<&ExplanationFinding>,
) -> ExplanationVerdict {
    if status_mismatch || return_mismatch || storage_mismatch {
        let mut fields = Vec::new();
        if status_mismatch {
            fields.push("status");
        }
        if return_mismatch {
            fields.push("return data");
        }
        if storage_mismatch {
            fields.push("storage");
        }
        let mismatch = format!("declared expectation differs for {}", fields.join(", "));
        return if root_cause.is_some() {
            ExplanationVerdict {
                code: "expectation-mismatch".into(),
                summary: format!(
                    "The {mismatch}; selected evidence establishes a causal mechanism."
                ),
            }
        } else {
            ExplanationVerdict {
                code: "insufficient-evidence".into(),
                summary: format!("The {mismatch}, but selected evidence does not establish why."),
            }
        };
    }
    if status == "revert" {
        return ExplanationVerdict {
            code: "execution-reverted".into(),
            summary: "The execution reverted and the trace identifies its terminal path.".into(),
        };
    }
    if status == "fault" {
        return ExplanationVerdict {
            code: "execution-faulted".into(),
            summary: "The execution faulted and the trace identifies its terminal path.".into(),
        };
    }
    if let Some(cause) = root_cause {
        return ExplanationVerdict {
            code: "completed-with-causal-findings".into(),
            summary: format!(
                "Execution completed, with causal finding: {}",
                cause.summary
            ),
        };
    }
    ExplanationVerdict {
        code: "execution-completed".into(),
        summary:
            "Execution completed and no failure mechanism was established by the selected evidence."
                .into(),
    }
}

fn choose_root_cause(
    findings: &[ExplanationFinding],
    status_mismatch: bool,
    return_mismatch: bool,
    storage_mismatch: bool,
    status: &str,
) -> Option<ExplanationFinding> {
    let mut priorities = Vec::new();
    if status_mismatch {
        priorities.extend([
            "child-frame-failure-continued",
            "execution-revert",
            "execution-fault",
            "rolled-back-write",
        ]);
    }
    if return_mismatch {
        priorities.extend([
            "child-frame-failure-continued",
            "storage-output-provenance",
            "arithmetic-input-provenance",
            "delegatecall-context-write",
        ]);
    }
    if storage_mismatch {
        priorities.extend([
            "delegatecall-context-write",
            "storage-write",
            "rolled-back-write",
        ]);
    }
    if priorities.is_empty() && (status == "revert" || status == "fault") {
        priorities.extend(["rolled-back-write", "execution-revert", "execution-fault"]);
    } else if priorities.is_empty() {
        priorities.extend([
            "child-frame-failure-continued",
            "delegatecall-context-write",
        ]);
    }
    priorities.iter().find_map(|code| {
        findings
            .iter()
            .find(|finding| finding.code == *code)
            .cloned()
    })
}

fn storage_matches(expectation: &ExplanationExpectation, execution: &Value) -> bool {
    let actual = execution.get("storage").and_then(Value::as_object);
    expectation.storage.iter().all(|(slot, expected)| {
        let observed = actual
            .and_then(|storage| storage.get(slot))
            .and_then(Value::as_str)
            .unwrap_or("0x0");
        normalize_storage_word(observed) == normalize_storage_word(expected)
    })
}

fn normalize_storage_word(value: &str) -> String {
    let stripped = value
        .trim()
        .trim_start_matches("0x")
        .trim_start_matches('0');
    if stripped.is_empty() {
        "0".into()
    } else {
        stripped.to_ascii_lowercase()
    }
}

fn usize_field(value: &Value, key: &str) -> usize {
    value.get(key).and_then(Value::as_u64).unwrap_or_default() as usize
}

fn return_data_matches(expected: &str, actual: &str) -> bool {
    let decode = |value: &str| hex::decode(value.trim().trim_start_matches("0x"));
    let (Ok(expected), Ok(actual)) = (decode(expected), decode(actual)) else {
        return false;
    };
    expected == actual
        || (expected.len() <= 32
            && actual.len() == 32
            && actual[..32 - expected.len()].iter().all(|byte| *byte == 0)
            && actual[32 - expected.len()..] == expected)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn identifies_continued_child_revert_as_root_cause() {
        let evidence = json!({
            "schema": "echoevm.evidence.v1",
            "profile": "revert",
            "execution": {"status": "success", "returnData": "0x", "totalSteps": 4},
            "events": [
                {"step": 0, "depth": 0, "pc": 10, "op": "CALL", "error": null},
                {"step": 1, "depth": 1, "pc": 20, "op": "SSTORE", "error": null},
                {"step": 2, "depth": 1, "pc": 30, "op": "REVERT", "reverted": true, "error": "Revert"}
            ],
            "links": [
                {"kind": "returns-to", "from": {"step": 2}, "to": {"step": 0}},
                {"kind": "rolls-back", "from": {"step": 1}, "to": {"step": 2}}
            ],
            "selection": {"candidates": 3, "selected": 3, "omitted": 0, "truncated": false}
        });
        let result = explain_evidence(&evidence, json!({"kind": "test"}), Default::default());
        assert_eq!(result.verdict.code, "completed-with-causal-findings");
        assert_eq!(
            result.root_cause.unwrap().code,
            "child-frame-failure-continued"
        );
    }

    #[test]
    fn fails_closed_when_expectation_differs_without_causal_links() {
        let evidence = json!({
            "schema": "echoevm.evidence.v1",
            "profile": "auto",
            "execution": {"status": "success", "returnData": "0x02", "totalSteps": 1},
            "events": [], "links": [],
            "selection": {"candidates": 0, "selected": 0, "omitted": 0, "truncated": false}
        });
        let expectation = ExplanationExpectation {
            status: None,
            return_data: Some("0x01".into()),
            storage: Default::default(),
        };
        let result = explain_evidence(&evidence, json!({"kind": "test"}), expectation);
        assert_eq!(result.verdict.code, "insufficient-evidence");
        assert!(result.root_cause.is_none());
    }

    #[test]
    fn uses_divisor_provenance_for_return_mismatch() {
        let evidence = json!({
            "schema": "echoevm.evidence.v1",
            "profile": "arithmetic",
            "execution": {"status": "success", "returnData": "0x06", "totalSteps": 2},
            "events": [
                {"step": 0, "depth": 0, "pc": 591, "op": "SUB"},
                {"step": 1, "depth": 0, "pc": 701, "op": "DIV"}
            ],
            "links": [{
                "kind": "value-flow", "input": 1, "value": "0x03",
                "from": {"step": 0}, "to": {"step": 1}
            }],
            "selection": {"candidates": 2, "selected": 2, "omitted": 0, "truncated": false}
        });
        let expectation = ExplanationExpectation {
            status: None,
            return_data: Some("0x05".into()),
            storage: Default::default(),
        };
        let result = explain_evidence(&evidence, json!({"kind": "test"}), expectation);
        assert_eq!(result.verdict.code, "expectation-mismatch");
        assert_eq!(
            result.root_cause.unwrap().code,
            "arithmetic-input-provenance"
        );
    }

    #[test]
    fn points_storage_mismatch_to_committed_write() {
        let slot = format!("0x{:064x}", 0);
        let evidence = json!({
            "schema": "echoevm.evidence.v1",
            "profile": "storage",
            "execution": {
                "status": "success",
                "returnData": "0x",
                "storage": {slot.clone(): format!("0x{:064x}", 42)},
                "totalSteps": 1
            },
            "events": [{
                "step": 0, "depth": 0, "pc": 4, "op": "SSTORE",
                "storage": [{
                    "kind": "write", "slot": slot.clone(),
                    "value": format!("0x{:064x}", 42)
                }]
            }],
            "links": [],
            "selection": {"candidates": 1, "selected": 1, "omitted": 0, "truncated": false}
        });
        let expectation = ExplanationExpectation {
            storage: [(slot, format!("0x{:064x}", 7))].into(),
            ..Default::default()
        };
        let result = explain_evidence(&evidence, json!({"kind": "test"}), expectation);
        assert_eq!(result.verdict.code, "expectation-mismatch");
        assert_eq!(result.root_cause.unwrap().code, "storage-write");
    }
}
