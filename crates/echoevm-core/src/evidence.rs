use super::*;
use std::collections::BTreeSet;

mod causal;

use causal::semantic_links;

/// Builds a deterministic, bounded execution-evidence view from an exact trace.
/// The limit only affects presentation; execution has already completed.
pub fn build_evidence(result: &WireResult, profile: &str, limit: usize) -> serde_json::Value {
    let steps = result.trace.as_deref().unwrap_or_default();
    let links: Vec<_> = semantic_links(steps)
        .into_iter()
        .filter(|link| {
            link.get("kind").and_then(|value| value.as_str()) != Some("value-flow")
                || matches!(profile, "arithmetic" | "full")
        })
        .collect();
    let linked_steps: BTreeSet<usize> = links
        .iter()
        .flat_map(|link| {
            [
                link.get("from")
                    .and_then(|value| value.get("step"))
                    .and_then(|value| value.as_u64()),
                link.get("to")
                    .and_then(|value| value.get("step"))
                    .and_then(|value| value.as_u64()),
            ]
        })
        .flatten()
        .map(|step| step as usize)
        .collect();
    let mut candidates: Vec<&TraceStep> = steps
        .iter()
        .filter(|step| evidence_selects(profile, step) || linked_steps.contains(&step.index))
        .collect();
    let candidate_count = candidates.len();
    let truncated = limit > 0 && candidates.len() > limit;
    if truncated {
        candidates.sort_by_key(|step| {
            (
                std::cmp::Reverse(evidence_priority_with_links(step, &links)),
                step.index,
            )
        });
        candidates.truncate(limit);
        candidates.sort_by_key(|step| step.index);
    }
    let selected_steps: BTreeSet<usize> = candidates.iter().map(|step| step.index).collect();
    let links: Vec<_> = links
        .into_iter()
        .filter(|link| {
            ["from", "to"].iter().all(|side| {
                link.get(side)
                    .and_then(|value| value.get("step"))
                    .and_then(|value| value.as_u64())
                    .is_some_and(|step| selected_steps.contains(&(step as usize)))
            })
        })
        .collect();
    let events: Vec<_> = candidates
        .iter()
        .map(|step| {
            json!({
                "step": step.index,
                "depth": step.depth,
                "address": step.address,
                "pc": step.pc,
                "op": step.opcode_name,
                "gas": {
                    "before": step.gas_before,
                    "after": step.gas_after,
                    "used": step.gas_before.saturating_sub(step.gas_after),
                    "staticCost": step.gas_before.saturating_sub(step.gas_after)
                },
                "stack": { "before": step.stack_before, "after": step.stack_after },
                "storage": step.storage,
                "control": step.control,
                "halt": matches!(step.opcode_name.as_str(), "STOP" | "RETURN" | "REVERT" | "INVALID" | "SELFDESTRUCT"),
                "reverted": step.opcode_name == "REVERT",
                "error": step.halt_class,
                "why": evidence_explanation(step)
            })
        })
        .collect();
    json!({
        "schema": echoevm_protocol::EVIDENCE_SCHEMA,
        "profile": profile,
        "execution": {
            "status": format!("{:?}", result.status).to_lowercase(),
            "gasUsed": result.gas_used,
            "returnData": result.return_data,
            "totalSteps": steps.len(),
            "error": result.error
        },
        "events": events,
        "links": links,
        "selection": {
            "candidates": candidate_count,
            "selected": candidates.len(),
            "omitted": candidate_count.saturating_sub(candidates.len()),
            "truncated": truncated
        }
    })
}

fn evidence_priority_with_links(step: &TraceStep, links: &[serde_json::Value]) -> u8 {
    let feeds_division = links.iter().any(|link| {
        link.get("kind").and_then(|value| value.as_str()) == Some("value-flow")
            && link
                .get("to")
                .and_then(|value| value.get("op"))
                .and_then(|value| value.as_str())
                .is_some_and(|op| matches!(op, "DIV" | "SDIV" | "MOD" | "SMOD"))
            && ["from", "to"].iter().any(|side| {
                link.get(side)
                    .and_then(|value| value.get("step"))
                    .and_then(|value| value.as_u64())
                    == Some(step.index as u64)
            })
    });
    if feeds_division {
        6
    } else {
        evidence_priority(step)
    }
}

fn evidence_selects(profile: &str, step: &TraceStep) -> bool {
    let op = step.opcode_name.as_str();
    match profile {
        "full" => true,
        "revert" => {
            matches!(op, "REVERT" | "INVALID" | "RETURN" | "STOP") || step.halt_class.is_some()
        }
        "storage" => matches!(op, "SLOAD" | "SSTORE" | "TLOAD" | "TSTORE"),
        "call" => matches!(
            op,
            "CALL"
                | "CALLCODE"
                | "DELEGATECALL"
                | "STATICCALL"
                | "CREATE"
                | "CREATE2"
                | "RETURN"
                | "REVERT"
        ),
        "abi" => matches!(
            op,
            "CALLDATALOAD"
                | "CALLDATACOPY"
                | "CALLDATASIZE"
                | "MLOAD"
                | "MSTORE"
                | "MSTORE8"
                | "RETURN"
                | "REVERT"
        ),
        "gas" => {
            step.gas_before.saturating_sub(step.gas_after) >= 100
                || matches!(op, "SSTORE" | "CALL" | "CREATE" | "CREATE2")
        }
        "arithmetic" => matches!(
            op,
            "ADD"
                | "SUB"
                | "MUL"
                | "DIV"
                | "SDIV"
                | "MOD"
                | "SMOD"
                | "ADDMOD"
                | "MULMOD"
                | "EXP"
                | "SIGNEXTEND"
        ),
        _ => {
            matches!(
                op,
                "REVERT"
                    | "INVALID"
                    | "SLOAD"
                    | "SSTORE"
                    | "TLOAD"
                    | "TSTORE"
                    | "CALL"
                    | "DELEGATECALL"
                    | "STATICCALL"
                    | "CREATE"
                    | "CREATE2"
                    | "RETURN"
                    | "SELFDESTRUCT"
            ) || step.halt_class.is_some()
        }
    }
}

fn evidence_priority(step: &TraceStep) -> u8 {
    if step.halt_class.is_some() || step.opcode_name == "REVERT" {
        5
    } else if matches!(step.opcode_name.as_str(), "SSTORE" | "TSTORE") {
        4
    } else if matches!(
        step.opcode_name.as_str(),
        "CALL" | "DELEGATECALL" | "STATICCALL" | "CREATE" | "CREATE2"
    ) {
        3
    } else {
        1
    }
}

fn evidence_explanation(step: &TraceStep) -> String {
    match step.opcode_name.as_str() {
        "SLOAD" => "Reads persistent contract storage.".into(),
        "SSTORE" => "Writes persistent contract storage if the frame commits.".into(),
        "TLOAD" | "TSTORE" => "Accesses transaction-scoped transient storage.".into(),
        "CALL" | "CALLCODE" | "DELEGATECALL" | "STATICCALL" => {
            "Transfers execution into another call frame.".into()
        }
        "CREATE" | "CREATE2" => "Creates a contract from initialization code.".into(),
        "REVERT" => "Reverts this frame and rolls back its state changes.".into(),
        "RETURN" => "Returns data successfully from this frame.".into(),
        op => format!("Executes {op} at program counter {}.", step.pc),
    }
}
