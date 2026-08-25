use super::{AbstractState, AbstractValue, Instruction};
use crate::opcode;
use alloy_primitives::U256;

pub(super) fn execute(instruction: &Instruction, state: &mut AbstractState) {
    let op = instruction.op;
    if op == 0x5f {
        state.push(AbstractValue::Constant(U256::ZERO));
        return;
    }
    if (0x60..=0x7f).contains(&op) {
        state.push(AbstractValue::Constant(U256::from_be_slice(
            &instruction.immediate,
        )));
        return;
    }
    if (0x80..=0x8f).contains(&op) {
        let depth = usize::from(op - 0x7f);
        let value = state
            .stack
            .len()
            .checked_sub(depth)
            .and_then(|index| state.stack.get(index))
            .cloned()
            .unwrap_or(AbstractValue::Unknown);
        state.push(value);
        return;
    }
    if (0x90..=0x9f).contains(&op) {
        let depth = usize::from(op - 0x8f);
        if state.stack.len() > depth {
            let top = state.stack.len() - 1;
            state.stack.swap(top, top - depth);
        }
        return;
    }

    match op {
        0x01..=0x07 | 0x0a..=0x14 | 0x16..=0x18 | 0x1a..=0x1d => {
            binary(state, opcode::name(op).unwrap_or("op"))
        }
        0x08 | 0x09 => ternary(state, opcode::name(op).unwrap_or("op")),
        0x15 | 0x19 | 0x1e => unary(state, opcode::name(op).unwrap_or("op")),
        0x20 => hash_memory(state),
        0x30 => state.push(AbstractValue::expression("address(this)")),
        0x31 => {
            state.pop();
            state.push(AbstractValue::expression("balance(account)"));
        }
        0x32 => state.push(AbstractValue::expression("tx.origin")),
        0x33 => state.push(AbstractValue::expression("caller")),
        0x34 => state.push(AbstractValue::expression("callvalue")),
        0x35 => load_calldata(state),
        0x36 => state.push(AbstractValue::expression("calldata.size")),
        0x37 | 0x39 | 0x3c | 0x3e | 0x5e => pop_n(state, 3),
        0x38 => state.push(AbstractValue::expression("code.size")),
        0x3a => state.push(AbstractValue::expression("gas.price")),
        0x3b | 0x3f => {
            state.pop();
            state.push(AbstractValue::expression("external.code"));
        }
        0x3d => state.push(AbstractValue::expression("returndata.size")),
        0x40 => {
            state.pop();
            state.push(AbstractValue::expression("block.hash"));
        }
        0x41 => state.push(AbstractValue::expression("block.coinbase")),
        0x42 => state.push(AbstractValue::expression("block.timestamp")),
        0x43 => state.push(AbstractValue::expression("block.number")),
        0x44 => state.push(AbstractValue::expression("block.prevrandao")),
        0x45 => state.push(AbstractValue::expression("block.gaslimit")),
        0x46 => state.push(AbstractValue::expression("chain.id")),
        0x47 => state.push(AbstractValue::expression("balance(this)")),
        0x48 => state.push(AbstractValue::expression("block.basefee")),
        0x49 => {
            state.pop();
            state.push(AbstractValue::expression("blob.hash"));
        }
        0x4a => state.push(AbstractValue::expression("blob.basefee")),
        0x50 => {
            state.pop();
        }
        0x51 => load_memory(state),
        0x52 => store_memory(state),
        0x53 => pop_n(state, 2),
        0x54 => load_storage(state),
        0x55 | 0x5d => pop_n(state, 2),
        0x58 => state.push(AbstractValue::Constant(U256::from(instruction.pc))),
        0x59 => state.push(AbstractValue::expression("memory.size")),
        0x5a => state.push(AbstractValue::expression("gas.remaining")),
        0x5b => {}
        0x5c => load_transient(state),
        0xa0..=0xa4 => pop_n(state, 2 + usize::from(op - 0xa0)),
        0xf0 => result_after_pop(state, 3, "created.address"),
        0xf1 | 0xf2 => result_after_pop(state, 7, "call.success"),
        0xf4 | 0xfa => result_after_pop(state, 6, "call.success"),
        0xf5 => result_after_pop(state, 4, "created.address"),
        0xf7 => {
            state.pop();
            state.push(AbstractValue::expression("returndata.word"));
        }
        0xf8 => result_after_pop(state, 4, "call.success"),
        0xf9 | 0xfb => result_after_pop(state, 3, "call.success"),
        0xff => {
            state.pop();
        }
        _ => {}
    }
}

pub(super) fn is_terminal(op: u8) -> bool {
    matches!(op, 0x00 | 0xf3 | 0xfd | 0xfe | 0xff)
}

fn hash_memory(state: &mut AbstractState) {
    let offset = state.pop();
    let size = state.pop();
    let detail = match (offset.constant_u64(), size.constant_u64()) {
        (Some(offset), Some(size)) if size <= 128 => {
            let words = (0..size.div_ceil(32))
                .map(|word| {
                    state
                        .memory
                        .get(&(offset + word * 32))
                        .cloned()
                        .unwrap_or(AbstractValue::Unknown)
                        .describe()
                })
                .collect::<Vec<_>>()
                .join(",");
            format!("keccak({words})")
        }
        _ => format!("keccak({},{})", offset.describe(), size.describe()),
    };
    state.push(AbstractValue::expression(detail));
}

fn load_calldata(state: &mut AbstractState) {
    let offset = state.pop();
    let description = offset.constant_u64().map_or_else(
        || format!("calldata[{}]", offset.describe()),
        |offset| {
            if offset >= 4 && (offset - 4) % 32 == 0 {
                format!("calldata.arg{}", (offset - 4) / 32)
            } else {
                format!("calldata.word@0x{offset:x}")
            }
        },
    );
    state.push(AbstractValue::expression(description));
}

fn load_memory(state: &mut AbstractState) {
    let offset = state.pop();
    let value = offset
        .constant_u64()
        .and_then(|offset| state.memory.get(&offset).cloned())
        .unwrap_or_else(|| AbstractValue::expression(format!("memory[{}]", offset.describe())));
    state.push(value);
}

fn store_memory(state: &mut AbstractState) {
    let offset = state.pop();
    let value = state.pop();
    if let Some(offset) = offset.constant_u64() {
        state.memory.insert(offset, value);
    }
}

fn load_storage(state: &mut AbstractState) {
    let slot = state.pop();
    state.push(AbstractValue::expression(format!(
        "storage[{}]",
        slot.describe()
    )));
}

fn load_transient(state: &mut AbstractState) {
    let slot = state.pop();
    state.push(AbstractValue::expression(format!(
        "transient[{}]",
        slot.describe()
    )));
}

fn result_after_pop(state: &mut AbstractState, count: usize, result: &str) {
    pop_n(state, count);
    state.push(AbstractValue::expression(result));
}

fn unary(state: &mut AbstractState, name: &str) {
    let value = state.pop();
    state.push(AbstractValue::expression(format!(
        "{}({})",
        name.to_ascii_lowercase(),
        value.describe()
    )));
}

fn binary(state: &mut AbstractState, name: &str) {
    let left = state.pop();
    let right = state.pop();
    state.push(AbstractValue::expression(format!(
        "{}({},{})",
        name.to_ascii_lowercase(),
        left.describe(),
        right.describe()
    )));
}

fn ternary(state: &mut AbstractState, name: &str) {
    let first = state.pop();
    let second = state.pop();
    let third = state.pop();
    state.push(AbstractValue::expression(format!(
        "{}({},{},{})",
        name.to_ascii_lowercase(),
        first.describe(),
        second.describe(),
        third.describe()
    )));
}

fn pop_n(state: &mut AbstractState, count: usize) {
    for _ in 0..count {
        state.pop();
    }
}
