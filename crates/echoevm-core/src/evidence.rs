use super::*;

/// Builds a deterministic, bounded execution-evidence view from an exact trace.
/// The limit only affects presentation; execution has already completed.
pub fn build_evidence(result: &WireResult, profile: &str, limit: usize) -> serde_json::Value {
    let steps = result.trace.as_deref().unwrap_or_default();
    let mut candidates: Vec<&TraceStep> = steps
        .iter()
        .filter(|step| evidence_selects(profile, step))
        .collect();
    let candidate_count = candidates.len();
    let truncated = limit > 0 && candidates.len() > limit;
    if truncated {
        candidates.sort_by_key(|step| (std::cmp::Reverse(evidence_priority(step)), step.index));
        candidates.truncate(limit);
        candidates.sort_by_key(|step| step.index);
    }
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
        "links": [],
        "selection": {
            "candidates": candidate_count,
            "selected": candidates.len(),
            "omitted": candidate_count.saturating_sub(candidates.len()),
            "truncated": truncated
        }
    })
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
