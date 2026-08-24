use super::*;
use std::collections::BTreeMap;

pub(super) fn semantic_links(steps: &[TraceStep]) -> Vec<serde_json::Value> {
    let mut links = Vec::new();
    for (position, step) in steps.iter().enumerate() {
        if !matches!(
            step.opcode_name.as_str(),
            "CALL" | "CALLCODE" | "DELEGATECALL" | "STATICCALL" | "CREATE" | "CREATE2"
        ) {
            continue;
        }
        let Some(first) = steps
            .get(position + 1)
            .filter(|child| child.depth == step.depth + 1)
        else {
            continue;
        };
        let end = steps[position + 1..]
            .iter()
            .position(|candidate| candidate.depth <= step.depth)
            .map_or(steps.len(), |offset| position + 1 + offset);
        let descendants = &steps[position + 1..end];
        let Some(last) = descendants.last() else {
            continue;
        };
        links.push(link("enters-frame", step, first, None, None));
        links.push(link("returns-to", last, step, None, None));
        let rolled_back = last.opcode_name == "REVERT"
            || last
                .halt_class
                .as_deref()
                .is_some_and(|halt| !matches!(halt, "Stop" | "Return"));
        if rolled_back {
            for write in descendants.iter().filter(|candidate| {
                candidate
                    .storage
                    .iter()
                    .any(|access| access.kind == "write")
            }) {
                links.push(link("rolls-back", write, last, None, None));
            }
        }
    }

    links.extend(value_flow_links(steps));
    links
}

fn value_flow_links(steps: &[TraceStep]) -> Vec<serde_json::Value> {
    let mut links = Vec::new();
    let mut stacks: BTreeMap<usize, Vec<Option<usize>>> = BTreeMap::new();
    let mut memory: BTreeMap<(usize, usize), Option<usize>> = BTreeMap::new();
    for step in steps {
        let origins = stacks.entry(step.depth).or_default();
        if origins.len() != step.stack_before.len() {
            origins.resize(step.stack_before.len(), None);
        }

        if is_arithmetic(&step.opcode_name) {
            for input in 0..arithmetic_inputs(&step.opcode_name) {
                let Some(stack_index) = origins.len().checked_sub(input + 1) else {
                    continue;
                };
                let Some(producer_index) = origins[stack_index] else {
                    continue;
                };
                let Some(producer) = steps.get(producer_index) else {
                    continue;
                };
                let value = step.stack_before.get(stack_index);
                links.push(link("value-flow", producer, step, Some(input), value));
            }
        }

        let memory_origin = match step.opcode_name.as_str() {
            "MSTORE" | "MSTORE8" => {
                let offset = stack_word_usize(&step.stack_before, 0);
                let value_origin = origins
                    .len()
                    .checked_sub(2)
                    .and_then(|index| origins[index]);
                if let Some(offset) = offset {
                    memory.insert((step.depth, offset), value_origin);
                }
                None
            }
            "MLOAD" => stack_word_usize(&step.stack_before, 0)
                .and_then(|offset| memory.get(&(step.depth, offset)).copied())
                .flatten(),
            _ => None,
        };

        let after = step.stack_after.as_deref().unwrap_or(&step.stack_before);
        match step.opcode_name.as_str() {
            op if op.starts_with("DUP") => {
                let depth = op[3..].parse::<usize>().unwrap_or_default();
                let origin = origins
                    .len()
                    .checked_sub(depth)
                    .and_then(|index| origins.get(index))
                    .copied()
                    .flatten();
                origins.push(origin);
            }
            op if op.starts_with("SWAP") => {
                let depth = op[4..].parse::<usize>().unwrap_or_default();
                if origins.len() > depth {
                    let top = origins.len() - 1;
                    origins.swap(top, top - depth);
                }
            }
            _ => {
                let prefix = common_prefix_len(&step.stack_before, after);
                origins.truncate(prefix);
                let pushed = after.len().saturating_sub(prefix);
                for _ in 0..pushed {
                    origins.push(if step.opcode_name == "MLOAD" {
                        memory_origin
                    } else {
                        Some(step.index)
                    });
                }
            }
        }
        if origins.len() != after.len() {
            origins.resize(after.len(), None);
        }
    }
    links
}

fn common_prefix_len(before: &[String], after: &[String]) -> usize {
    before
        .iter()
        .zip(after)
        .take_while(|(left, right)| left == right)
        .count()
}

fn stack_word_usize(stack: &[String], from_top: usize) -> Option<usize> {
    let index = stack.len().checked_sub(from_top + 1)?;
    let value = stack.get(index)?.strip_prefix("0x")?;
    usize::from_str_radix(value, 16).ok()
}

fn link(
    kind: &str,
    from: &TraceStep,
    to: &TraceStep,
    input: Option<usize>,
    value: Option<&String>,
) -> serde_json::Value {
    let mut result = json!({
        "kind": kind,
        "from": {"step": from.index, "depth": from.depth, "pc": from.pc, "op": from.opcode_name},
        "to": {"step": to.index, "depth": to.depth, "pc": to.pc, "op": to.opcode_name}
    });
    if let Some(object) = result.as_object_mut() {
        if let Some(input) = input {
            object.insert("input".into(), json!(input));
        }
        if let Some(value) = value {
            object.insert("value".into(), json!(value));
        }
    }
    result
}

fn is_arithmetic(op: &str) -> bool {
    matches!(
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
    )
}

fn arithmetic_inputs(op: &str) -> usize {
    match op {
        "ADDMOD" | "MULMOD" => 3,
        "EXP" | "SIGNEXTEND" | "ADD" | "SUB" | "MUL" | "DIV" | "SDIV" | "MOD" | "SMOD" => 2,
        _ => 0,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use echoevm_protocol::StorageAccess;

    fn step(
        index: usize,
        depth: usize,
        pc: u64,
        op: &str,
        before: &[u64],
        after: &[u64],
    ) -> TraceStep {
        TraceStep {
            index,
            depth,
            address: Some(format!("0x{:040x}", depth + 1)),
            pc,
            opcode: "0x00".into(),
            opcode_name: op.into(),
            gas_before: 100,
            gas_after: 97,
            stack_before: before
                .iter()
                .map(|value| format!("0x{value:064x}"))
                .collect(),
            stack_after: Some(
                after
                    .iter()
                    .map(|value| format!("0x{value:064x}"))
                    .collect(),
            ),
            halt_class: None,
            storage: Vec::new(),
            control: None,
        }
    }

    #[test]
    fn links_child_revert_to_call_and_rolled_back_write() {
        let call = step(0, 0, 10, "CALL", &[], &[0]);
        let mut write = step(1, 1, 20, "SSTORE", &[42, 0], &[]);
        write.storage.push(StorageAccess {
            kind: "write".into(),
            address: "0x0000000000000000000000000000000000000002".into(),
            slot: "0x00".into(),
            previous: Some("0x00".into()),
            value: Some("0x2a".into()),
        });
        let revert = step(2, 1, 30, "REVERT", &[0, 0], &[]);
        let parent = step(3, 0, 11, "STOP", &[], &[]);
        let links = semantic_links(&[call, write, revert, parent]);

        assert!(links.iter().any(|link| link["kind"] == "enters-frame"));
        assert!(links.iter().any(|link| {
            link["kind"] == "returns-to" && link["from"]["pc"] == 30 && link["to"]["pc"] == 10
        }));
        assert!(links.iter().any(|link| {
            link["kind"] == "rolls-back" && link["from"]["pc"] == 20 && link["to"]["pc"] == 30
        }));
    }

    #[test]
    fn preserves_arithmetic_origin_through_dup_and_swap() {
        let steps = vec![
            step(0, 0, 10, "SUB", &[5, 2], &[3]),
            step(1, 0, 11, "DUP1", &[3], &[3, 3]),
            step(2, 0, 12, "PUSH1", &[3, 3], &[3, 3, 20]),
            step(3, 0, 13, "SWAP1", &[3, 3, 20], &[3, 20, 3]),
            step(4, 0, 14, "DIV", &[3, 20, 3], &[3, 6]),
        ];
        let links = value_flow_links(&steps);
        assert!(links.iter().any(|link| {
            link["kind"] == "value-flow"
                && link["from"]["pc"] == 10
                && link["to"]["pc"] == 14
                && link["value"] == format!("0x{:064x}", 3)
        }));
    }
}
