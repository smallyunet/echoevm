use crate::{ENGINE_NAME, ENGINE_VERSION, opcode};
use alloy_primitives::U256;
use echoevm_protocol::{
    BEHAVIOR_SCHEMA, BehaviorBytecode, BehaviorCoverage, BehaviorDocument, BehaviorEffect,
    BehaviorFunction, BehaviorGuard,
};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet, HashMap, VecDeque};

mod abstract_exec;

const MAX_STATES_PER_PC: usize = 8;
const MAX_STEPS_PER_ENTRY: usize = 20_000;
const MAX_EXPRESSION_BYTES: usize = 160;

#[derive(Clone, Debug)]
struct Instruction {
    pc: usize,
    op: u8,
    immediate: Vec<u8>,
    next: usize,
}

#[derive(Clone, Debug, PartialEq, Eq)]
enum AbstractValue {
    Unknown,
    Constant(U256),
    Expression(String),
}

impl AbstractValue {
    fn describe(&self) -> String {
        match self {
            Self::Unknown => "unknown".into(),
            Self::Constant(value) => format!("constant(0x{value:x})"),
            Self::Expression(value) => value.clone(),
        }
    }

    fn constant_u64(&self) -> Option<u64> {
        let Self::Constant(value) = self else {
            return None;
        };
        (*value <= U256::from(u64::MAX)).then(|| value.to::<u64>())
    }

    fn expression(value: impl Into<String>) -> Self {
        let mut value = value.into();
        if value.len() > MAX_EXPRESSION_BYTES {
            value.truncate(MAX_EXPRESSION_BYTES - 1);
            value.push('…');
        }
        Self::Expression(value)
    }
}

#[derive(Clone, Debug, Default)]
struct AbstractState {
    stack: Vec<AbstractValue>,
    memory: BTreeMap<u64, AbstractValue>,
}

impl AbstractState {
    fn pop(&mut self) -> AbstractValue {
        self.stack.pop().unwrap_or(AbstractValue::Unknown)
    }

    fn push(&mut self, value: AbstractValue) {
        if self.stack.len() >= 1_024 {
            self.stack.remove(0);
        }
        self.stack.push(value);
    }
}

#[derive(Default)]
struct EntryAnalysis {
    effects: Vec<BehaviorEffect>,
    guards: Vec<BehaviorGuard>,
    reachable: BTreeSet<usize>,
    unresolved_jumps: BTreeSet<usize>,
    truncated: bool,
}

pub fn infer_behavior(bytecode: &[u8]) -> BehaviorDocument {
    let instructions = decode(bytecode);
    let by_pc = instructions
        .iter()
        .cloned()
        .map(|instruction| (instruction.pc, instruction))
        .collect::<BTreeMap<_, _>>();
    let selectors = discover_selectors(&instructions);
    let unknown_opcodes = instructions
        .iter()
        .filter(|instruction| opcode::name(instruction.op).is_none())
        .count();

    let has_selectors = !selectors.is_empty();
    let mut entries = selectors;
    entries.push(("fallback".into(), 0usize));
    let mut functions = Vec::new();
    let mut all_reachable = BTreeSet::new();
    let mut all_unresolved = BTreeSet::new();
    let mut all_effects = BTreeMap::new();
    let mut contract_capabilities = BTreeSet::new();
    let mut truncated = false;

    for (selector, entry_pc) in entries {
        let mut analysis = analyze_entry(&by_pc, entry_pc);
        let recognized_selector = usize::from(selector != "fallback");
        all_reachable.extend(analysis.reachable.iter().copied());
        all_unresolved.extend(analysis.unresolved_jumps.iter().copied());
        truncated |= analysis.truncated;
        let discovered_effects = analysis.effects.clone();
        contract_capabilities.extend(capabilities(&discovered_effects));
        if selector == "fallback" && has_selectors {
            analysis
                .effects
                .retain(|effect| !all_effects.contains_key(&effect_key(effect)));
            analysis.guards.clear();
        }
        let capabilities = capabilities(&analysis.effects);
        for effect in &discovered_effects {
            all_effects
                .entry(effect_key(effect))
                .or_insert_with(|| effect.clone());
        }
        functions.push(BehaviorFunction {
            selector,
            signature: None,
            entry_pc: entry_pc as u64,
            capabilities,
            effects: analysis.effects,
            guards: analysis.guards,
            coverage: BehaviorCoverage {
                recognized_selectors: recognized_selector,
                analyzed_entry_points: 1,
                reachable_instructions: analysis.reachable.len(),
                unknown_opcodes,
                unresolved_jumps: analysis.unresolved_jumps.len(),
                truncated: analysis.truncated,
            },
        });
    }
    functions.sort_by(|left, right| left.selector.cmp(&right.selector));

    let selector_count = functions
        .iter()
        .filter(|function| function.selector != "fallback")
        .count();
    let mut limitations = vec![
        "Behavioral effects are inferred from runtime bytecode by bounded abstract execution; they are not a security audit or formal proof.".into(),
        "Dynamic jump and external-call destinations remain explicit unknowns when bytecode does not make them statically recoverable.".into(),
    ];
    if selector_count > 0 {
        limitations.push(
            "Selector recovery recognizes common PUSH4/EQ dispatchers. The fallback summary starts at the runtime entry and reports additional effects not already attributed to selector entries; dispatcher-wide guards are omitted from that summary.".into(),
        );
    }
    if truncated {
        limitations.push("At least one entry point reached the bounded exploration limit.".into());
    }

    BehaviorDocument {
        schema: BEHAVIOR_SCHEMA.into(),
        engine: ENGINE_NAME.into(),
        engine_version: ENGINE_VERSION.into(),
        bytecode: BehaviorBytecode {
            sha256: format!("0x{}", hex::encode(Sha256::digest(bytecode))),
            bytes: bytecode.len(),
            instructions: instructions.len(),
        },
        coverage: BehaviorCoverage {
            recognized_selectors: selector_count,
            analyzed_entry_points: functions.len(),
            reachable_instructions: all_reachable.len(),
            unknown_opcodes,
            unresolved_jumps: all_unresolved.len(),
            truncated,
        },
        functions,
        contract_capabilities: contract_capabilities.into_iter().collect(),
        contract_effects: all_effects.into_values().collect(),
        limitations,
    }
}

fn effect_key(effect: &BehaviorEffect) -> (u64, String, String) {
    (effect.pc, effect.kind.clone(), effect.opcode.clone())
}

fn decode(bytecode: &[u8]) -> Vec<Instruction> {
    let mut pc = 0usize;
    let mut instructions = Vec::new();
    while pc < bytecode.len() {
        let op = bytecode[pc];
        let width = if (0x60..=0x7f).contains(&op) {
            usize::from(op - 0x5f)
        } else {
            0
        };
        let next = (pc + 1 + width).min(bytecode.len());
        instructions.push(Instruction {
            pc,
            op,
            immediate: bytecode[pc + 1..next].to_vec(),
            next,
        });
        pc = next;
    }
    instructions
}

fn discover_selectors(instructions: &[Instruction]) -> Vec<(String, usize)> {
    let mut selectors = BTreeMap::new();
    for (index, instruction) in instructions.iter().enumerate() {
        if instruction.op != 0x63 || instruction.immediate.len() != 4 {
            continue;
        }
        let mut eq_index = None;
        for (candidate, next) in instructions
            .iter()
            .enumerate()
            .take(instructions.len().min(index + 5))
            .skip(index + 1)
        {
            if next.op == 0x63 {
                break;
            }
            if next.op == 0x14 {
                eq_index = Some(candidate);
                break;
            }
        }
        let Some(eq_index) = eq_index else {
            continue;
        };
        for push_index in (eq_index + 1)..instructions.len().min(eq_index + 5) {
            let push = &instructions[push_index];
            if !(0x60..=0x7f).contains(&push.op) {
                continue;
            }
            let Some(jump) = instructions.get(push_index + 1) else {
                continue;
            };
            if jump.op != 0x57 {
                continue;
            }
            let destination = U256::from_be_slice(&push.immediate);
            if destination <= U256::from(usize::MAX) {
                selectors.insert(
                    format!("0x{}", hex::encode(&instruction.immediate)),
                    destination.to::<usize>(),
                );
            }
            break;
        }
    }
    selectors.into_iter().collect()
}

fn analyze_entry(instructions: &BTreeMap<usize, Instruction>, entry_pc: usize) -> EntryAnalysis {
    let mut analysis = EntryAnalysis::default();
    let mut queue = VecDeque::from([(entry_pc, AbstractState::default())]);
    let mut visits = HashMap::<usize, usize>::new();
    let mut steps = 0usize;

    while let Some((pc, mut state)) = queue.pop_front() {
        if steps >= MAX_STEPS_PER_ENTRY {
            analysis.truncated = true;
            break;
        }
        let count = visits.entry(pc).or_default();
        if *count >= MAX_STATES_PER_PC {
            analysis.truncated = true;
            continue;
        }
        *count += 1;
        let Some(instruction) = instructions.get(&pc) else {
            continue;
        };
        steps += 1;
        analysis.reachable.insert(pc);

        if let Some(effect) = capture_effect(instruction, &state) {
            analysis.effects.push(effect);
        }

        match instruction.op {
            0x56 => {
                let destination = state.pop();
                if let Some(destination) = destination.constant_u64() {
                    queue.push_back((destination as usize, state));
                } else {
                    analysis.unresolved_jumps.insert(pc);
                }
                continue;
            }
            0x57 => {
                let destination = state.pop();
                let condition = state.pop();
                let target = destination.constant_u64();
                if is_semantic_guard(&condition) {
                    analysis.guards.push(BehaviorGuard {
                        pc: pc as u64,
                        condition: condition.describe(),
                        destination: target,
                    });
                }
                if let Some(destination) = target {
                    queue.push_back((destination as usize, state.clone()));
                } else {
                    analysis.unresolved_jumps.insert(pc);
                }
                queue.push_back((instruction.next, state));
                continue;
            }
            _ => execute_abstract(instruction, &mut state),
        }

        if !is_terminal(instruction.op) {
            queue.push_back((instruction.next, state));
        }
    }

    analysis
        .effects
        .sort_by_key(|effect| (effect.pc, effect.kind.clone()));
    analysis
        .effects
        .dedup_by(|left, right| left.pc == right.pc && left.kind == right.kind);
    analysis.guards.sort_by_key(|guard| guard.pc);
    analysis.guards.dedup_by(|left, right| left.pc == right.pc);
    analysis
}

fn capture_effect(instruction: &Instruction, state: &AbstractState) -> Option<BehaviorEffect> {
    let mut stack = state.clone();
    let (kind, inputs) = match instruction.op {
        0x31 => ("balance-read", fields([("account", stack.pop())])),
        0x3b | 0x3c | 0x3f => ("external-code-read", fields([("account", stack.pop())])),
        0x54 => ("storage-read", fields([("slot", stack.pop())])),
        0x55 => (
            "storage-write",
            fields([("slot", stack.pop()), ("value", stack.pop())]),
        ),
        0x5c => ("transient-read", fields([("slot", stack.pop())])),
        0x5d => (
            "transient-write",
            fields([("slot", stack.pop()), ("value", stack.pop())]),
        ),
        0xa0..=0xa4 => (
            "event-log",
            BTreeMap::from([(
                "topics".into(),
                usize::from(instruction.op - 0xa0).to_string(),
            )]),
        ),
        0xf0 => (
            "contract-create",
            fields([
                ("value", stack.pop()),
                ("initOffset", stack.pop()),
                ("initSize", stack.pop()),
            ]),
        ),
        0xf1 | 0xf2 => {
            let _gas = stack.pop();
            let target = stack.pop();
            let value = stack.pop();
            (
                if instruction.op == 0xf1 {
                    "external-call"
                } else {
                    "callcode"
                },
                fields([("target", target), ("value", value)]),
            )
        }
        0xf4 => {
            let _gas = stack.pop();
            ("delegate-call", fields([("target", stack.pop())]))
        }
        0xf5 => (
            "contract-create2",
            fields([
                ("value", stack.pop()),
                ("initOffset", stack.pop()),
                ("initSize", stack.pop()),
                ("salt", stack.pop()),
            ]),
        ),
        0xf8 => (
            "external-call",
            fields([("target", stack.pop()), ("value", stack.pop())]),
        ),
        0xf9 => ("delegate-call", fields([("target", stack.pop())])),
        0xfa | 0xfb => {
            let _gas = (instruction.op == 0xfa).then(|| stack.pop());
            ("static-call", fields([("target", stack.pop())]))
        }
        0xff => ("self-destruct", fields([("beneficiary", stack.pop())])),
        _ => return None,
    };
    Some(BehaviorEffect {
        kind: kind.into(),
        pc: instruction.pc as u64,
        opcode: opcode::name(instruction.op).unwrap_or("UNKNOWN").into(),
        inputs,
    })
}

fn fields<const N: usize>(values: [(&str, AbstractValue); N]) -> BTreeMap<String, String> {
    values
        .into_iter()
        .map(|(name, value)| (name.into(), value.describe()))
        .collect()
}

fn execute_abstract(instruction: &Instruction, state: &mut AbstractState) {
    abstract_exec::execute(instruction, state);
}

fn is_terminal(op: u8) -> bool {
    abstract_exec::is_terminal(op)
}
fn is_semantic_guard(condition: &AbstractValue) -> bool {
    let description = condition.describe();
    [
        "caller",
        "tx.origin",
        "callvalue",
        "storage[",
        "transient[",
        "block.",
    ]
    .iter()
    .any(|needle| description.contains(needle))
}

fn capabilities(effects: &[BehaviorEffect]) -> Vec<String> {
    effects
        .iter()
        .filter_map(|effect| {
            Some(
                match effect.kind.as_str() {
                    "storage-read" => "reads-persistent-state",
                    "storage-write" => "writes-persistent-state",
                    "transient-read" | "transient-write" => "uses-transient-state",
                    "external-call" | "callcode" => "calls-external-code",
                    "delegate-call" => "executes-delegate-code",
                    "static-call" => "reads-external-state",
                    "contract-create" | "contract-create2" => "creates-contracts",
                    "self-destruct" => "can-self-destruct",
                    "event-log" => "emits-events",
                    "balance-read" | "external-code-read" => "reads-external-state",
                    _ => return None,
                }
                .to_string(),
            )
        })
        .collect::<BTreeSet<_>>()
        .into_iter()
        .collect()
}

#[cfg(test)]
mod tests;
