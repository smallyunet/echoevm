use echoevm_protocol::{ExplanationFinding, ExplanationReference};
use serde_json::Value;

pub(super) fn collect(evidence: &Value, status: &str) -> Vec<ExplanationFinding> {
    let mut findings = Vec::new();
    findings.extend(child_failure_findings(evidence, status));
    findings.extend(rollback_findings(evidence));
    findings.extend(delegate_context_findings(evidence));
    findings.extend(arithmetic_findings(evidence));
    if matches!(status, "revert" | "fault")
        && !findings.iter().any(is_terminal_finding)
        && let Some(event) = events(evidence)
            .iter()
            .rev()
            .find(|event| event_is_failure(event))
    {
        findings.push(ExplanationFinding {
            code: format!("execution-{status}"),
            summary: format!("The top-level execution ended with status {status}."),
            basis: "execution status and terminal trace event".into(),
            confidence: "direct".into(),
            evidence: vec![event_reference(event, Some("terminal"))],
        });
    }
    findings
}

fn child_failure_findings(evidence: &Value, status: &str) -> Vec<ExplanationFinding> {
    if status != "success" {
        return Vec::new();
    }
    links(evidence)
        .iter()
        .filter(|link| link.get("kind").and_then(Value::as_str) == Some("returns-to"))
        .filter_map(|link| {
            let from = link.get("from")?;
            let event = event_by_step(evidence, usize_field(from, "step"))?;
            event_is_failure(event).then(|| ExplanationFinding {
                code: "child-frame-failure-continued".into(),
                summary: "A nested frame failed, returned to its caller, and top-level execution continued successfully.".into(),
                basis: "returns-to link, failed child terminal event, and successful top-level status".into(),
                confidence: "direct".into(),
                evidence: link_references(evidence, link),
            })
        })
        .collect()
}

fn rollback_findings(evidence: &Value) -> Vec<ExplanationFinding> {
    links(evidence)
        .iter()
        .filter(|link| link.get("kind").and_then(Value::as_str) == Some("rolls-back"))
        .map(|link| ExplanationFinding {
            code: "rolled-back-write".into(),
            summary: "A storage write occurred inside a frame that later reverted.".into(),
            basis: "rolls-back causal link between the storage write and frame terminal".into(),
            confidence: "direct".into(),
            evidence: link_references(evidence, link),
        })
        .collect()
}

fn delegate_context_findings(evidence: &Value) -> Vec<ExplanationFinding> {
    let all_events = events(evidence);
    all_events
        .iter()
        .filter(|event| event.get("op").and_then(Value::as_str) == Some("DELEGATECALL"))
        .filter_map(|call| {
            let call_step = usize_field(call, "step");
            let call_depth = usize_field(call, "depth");
            let address = call.get("address").and_then(Value::as_str)?;
            let entry = links(evidence).iter().find(|link| {
                link.get("kind").and_then(Value::as_str) == Some("enters-frame")
                    && link.get("from").is_some_and(|from| usize_field(from, "step") == call_step)
            })?;
            let first = entry.get("to").map(|to| usize_field(to, "step"))?;
            let last = links(evidence)
                .iter()
                .find(|link| {
                    link.get("kind").and_then(Value::as_str) == Some("returns-to")
                        && link.get("to").is_some_and(|to| usize_field(to, "step") == call_step)
                })
                .and_then(|link| link.get("from"))
                .map(|from| usize_field(from, "step"))?;
            let write = all_events.iter().find(|event| {
                let step = usize_field(event, "step");
                step >= first
                    && step <= last
                    && usize_field(event, "depth") > call_depth
                    && event.get("address").and_then(Value::as_str) == Some(address)
                    && event
                        .get("storage")
                        .and_then(Value::as_array)
                        .is_some_and(|accesses| {
                            accesses.iter().any(|access| {
                                access.get("kind").and_then(Value::as_str) == Some("write")
                                    && access.get("address").and_then(Value::as_str) == Some(address)
                            })
                        })
            })?;
            Some(ExplanationFinding {
                code: "delegatecall-context-write".into(),
                summary: "Code entered through DELEGATECALL wrote storage in the caller's execution context.".into(),
                basis: "delegate frame bounds and matching execution-context storage address".into(),
                confidence: "direct".into(),
                evidence: vec![
                    event_reference(call, Some("delegatecall")),
                    event_reference(write, Some("context-write")),
                ],
            })
        })
        .collect()
}

fn arithmetic_findings(evidence: &Value) -> Vec<ExplanationFinding> {
    let mut findings: Vec<_> = links(evidence)
        .iter()
        .filter(|link| link.get("kind").and_then(Value::as_str) == Some("value-flow"))
        .filter(|link| {
            link.get("to")
                .and_then(|to| endpoint_op(evidence, to))
                .is_some_and(|op| matches!(op, "DIV" | "SDIV" | "MOD" | "SMOD"))
        })
        .map(|link| {
            let value = link.get("value").and_then(Value::as_str).unwrap_or("unknown");
            let input = link.get("input").and_then(Value::as_u64).unwrap_or_default();
            (
                input,
                ExplanationFinding {
                    code: "arithmetic-input-provenance".into(),
                    summary: format!("A tracked arithmetic value {value} flowed into operand {input} of a division or modulo operation."),
                    basis: "tracked stack provenance in the captured execution".into(),
                    confidence: "direct".into(),
                    evidence: link_references(evidence, link),
                },
            )
        })
        .collect();
    findings.sort_by_key(|(input, _)| u8::from(*input != 1));
    findings.into_iter().map(|(_, finding)| finding).collect()
}

fn is_terminal_finding(finding: &ExplanationFinding) -> bool {
    matches!(
        finding.code.as_str(),
        "execution-revert" | "execution-fault"
    )
}

fn event_is_failure(event: &Value) -> bool {
    event
        .get("reverted")
        .and_then(Value::as_bool)
        .unwrap_or(false)
        || event
            .get("error")
            .and_then(Value::as_str)
            .is_some_and(|error| !matches!(error, "Stop" | "Return"))
        || event
            .get("op")
            .and_then(Value::as_str)
            .is_some_and(|op| op == "INVALID")
}

fn link_references(evidence: &Value, link: &Value) -> Vec<ExplanationReference> {
    ["from", "to"]
        .iter()
        .filter_map(|side| {
            let endpoint = link.get(side)?;
            let step = usize_field(endpoint, "step");
            event_by_step(evidence, step)
                .map(|event| event_reference(event, Some(side)))
                .or_else(|| endpoint_reference(endpoint, Some(side)))
        })
        .collect()
}

fn event_reference(event: &Value, relation: Option<&str>) -> ExplanationReference {
    ExplanationReference {
        step: usize_field(event, "step"),
        depth: usize_field(event, "depth"),
        pc: u64_field(event, "pc"),
        op: event
            .get("op")
            .and_then(Value::as_str)
            .unwrap_or("UNKNOWN")
            .into(),
        relation: relation.map(str::to_owned),
        source: event.get("source").cloned(),
    }
}

fn endpoint_reference(endpoint: &Value, relation: Option<&str>) -> Option<ExplanationReference> {
    Some(ExplanationReference {
        step: usize_field(endpoint, "step"),
        depth: usize_field(endpoint, "depth"),
        pc: u64_field(endpoint, "pc"),
        op: endpoint.get("op")?.as_str()?.into(),
        relation: relation.map(str::to_owned),
        source: None,
    })
}

fn event_by_step(evidence: &Value, step: usize) -> Option<&Value> {
    events(evidence)
        .iter()
        .find(|event| usize_field(event, "step") == step)
}

fn endpoint_op<'a>(evidence: &'a Value, endpoint: &'a Value) -> Option<&'a str> {
    endpoint.get("op").and_then(Value::as_str).or_else(|| {
        event_by_step(evidence, usize_field(endpoint, "step"))?
            .get("op")?
            .as_str()
    })
}

fn events(evidence: &Value) -> &[Value] {
    evidence
        .get("events")
        .and_then(Value::as_array)
        .map(Vec::as_slice)
        .unwrap_or_default()
}

fn links(evidence: &Value) -> &[Value] {
    evidence
        .get("links")
        .and_then(Value::as_array)
        .map(Vec::as_slice)
        .unwrap_or_default()
}

fn usize_field(value: &Value, key: &str) -> usize {
    value.get(key).and_then(Value::as_u64).unwrap_or_default() as usize
}

fn u64_field(value: &Value, key: &str) -> u64 {
    value.get(key).and_then(Value::as_u64).unwrap_or_default()
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn successful_child_return_is_not_a_failure() {
        let event = json!({"op": "RETURN", "reverted": false, "error": "Return"});
        assert!(!event_is_failure(&event));
    }
}
