//! EchoEVM-owned opcode metadata.

/// Returns the canonical mnemonic for an opcode byte known to EchoEVM.
pub const fn name(op: u8) -> Option<&'static str> {
    Some(match op {
        0x00 => "STOP",
        0x01 => "ADD",
        0x02 => "MUL",
        0x03 => "SUB",
        0x04 => "DIV",
        0x05 => "SDIV",
        0x06 => "MOD",
        0x07 => "SMOD",
        0x08 => "ADDMOD",
        0x09 => "MULMOD",
        0x0a => "EXP",
        0x0b => "SIGNEXTEND",
        0x10 => "LT",
        0x11 => "GT",
        0x12 => "SLT",
        0x13 => "SGT",
        0x14 => "EQ",
        0x15 => "ISZERO",
        0x16 => "AND",
        0x17 => "OR",
        0x18 => "XOR",
        0x19 => "NOT",
        0x1a => "BYTE",
        0x1b => "SHL",
        0x1c => "SHR",
        0x1d => "SAR",
        0x1e => "CLZ",
        0x20 => "KECCAK256",
        0x30 => "ADDRESS",
        0x31 => "BALANCE",
        0x32 => "ORIGIN",
        0x33 => "CALLER",
        0x34 => "CALLVALUE",
        0x35 => "CALLDATALOAD",
        0x36 => "CALLDATASIZE",
        0x37 => "CALLDATACOPY",
        0x38 => "CODESIZE",
        0x39 => "CODECOPY",
        0x3a => "GASPRICE",
        0x3b => "EXTCODESIZE",
        0x3c => "EXTCODECOPY",
        0x3d => "RETURNDATASIZE",
        0x3e => "RETURNDATACOPY",
        0x3f => "EXTCODEHASH",
        0x40 => "BLOCKHASH",
        0x41 => "COINBASE",
        0x42 => "TIMESTAMP",
        0x43 => "NUMBER",
        0x44 => "PREVRANDAO",
        0x45 => "GASLIMIT",
        0x46 => "CHAINID",
        0x47 => "SELFBALANCE",
        0x48 => "BASEFEE",
        0x49 => "BLOBHASH",
        0x4a => "BLOBBASEFEE",
        0x4b => "SLOTNUM",
        0x50 => "POP",
        0x51 => "MLOAD",
        0x52 => "MSTORE",
        0x53 => "MSTORE8",
        0x54 => "SLOAD",
        0x55 => "SSTORE",
        0x56 => "JUMP",
        0x57 => "JUMPI",
        0x58 => "PC",
        0x59 => "MSIZE",
        0x5a => "GAS",
        0x5b => "JUMPDEST",
        0x5c => "TLOAD",
        0x5d => "TSTORE",
        0x5e => "MCOPY",
        0x5f => "PUSH0",
        0x60..=0x7f => return push_name(op),
        0x80..=0x8f => return dup_name(op),
        0x90..=0x9f => return swap_name(op),
        0xa0 => "LOG0",
        0xa1 => "LOG1",
        0xa2 => "LOG2",
        0xa3 => "LOG3",
        0xa4 => "LOG4",
        0xd0 => "DATALOAD",
        0xd1 => "DATALOADN",
        0xd2 => "DATASIZE",
        0xd3 => "DATACOPY",
        0xe0 => "RJUMP",
        0xe1 => "RJUMPI",
        0xe2 => "RJUMPV",
        0xe3 => "CALLF",
        0xe4 => "RETF",
        0xe5 => "JUMPF",
        0xe6 => "DUPN",
        0xe7 => "SWAPN",
        0xe8 => "EXCHANGE",
        0xec => "EOFCREATE",
        0xee => "RETURNCONTRACT",
        0xf0 => "CREATE",
        0xf1 => "CALL",
        0xf2 => "CALLCODE",
        0xf3 => "RETURN",
        0xf4 => "DELEGATECALL",
        0xf5 => "CREATE2",
        0xf7 => "RETURNDATALOAD",
        0xf8 => "EXTCALL",
        0xf9 => "EXTDELEGATECALL",
        0xfb => "EXTSTATICCALL",
        0xfa => "STATICCALL",
        0xfd => "REVERT",
        0xfe => "INVALID",
        0xff => "SELFDESTRUCT",
        _ => return None,
    })
}

macro_rules! numbered_names {
    ($fn_name:ident, $base:expr, [$($name:literal),+ $(,)?]) => {
        const fn $fn_name(op: u8) -> Option<&'static str> {
            const NAMES: &[&str] = &[$($name),+];
            let index = op.wrapping_sub($base) as usize;
            if index < NAMES.len() { Some(NAMES[index]) } else { None }
        }
    };
}

numbered_names!(
    push_name,
    0x60,
    [
        "PUSH1", "PUSH2", "PUSH3", "PUSH4", "PUSH5", "PUSH6", "PUSH7", "PUSH8", "PUSH9", "PUSH10",
        "PUSH11", "PUSH12", "PUSH13", "PUSH14", "PUSH15", "PUSH16", "PUSH17", "PUSH18", "PUSH19",
        "PUSH20", "PUSH21", "PUSH22", "PUSH23", "PUSH24", "PUSH25", "PUSH26", "PUSH27", "PUSH28",
        "PUSH29", "PUSH30", "PUSH31", "PUSH32",
    ]
);
numbered_names!(
    dup_name,
    0x80,
    [
        "DUP1", "DUP2", "DUP3", "DUP4", "DUP5", "DUP6", "DUP7", "DUP8", "DUP9", "DUP10", "DUP11",
        "DUP12", "DUP13", "DUP14", "DUP15", "DUP16",
    ]
);
numbered_names!(
    swap_name,
    0x90,
    [
        "SWAP1", "SWAP2", "SWAP3", "SWAP4", "SWAP5", "SWAP6", "SWAP7", "SWAP8", "SWAP9", "SWAP10",
        "SWAP11", "SWAP12", "SWAP13", "SWAP14", "SWAP15", "SWAP16",
    ]
);

pub fn by_name(input: &str) -> Option<u8> {
    let upper = input.to_ascii_uppercase();
    (0..=u8::MAX).find(|op| name(*op).is_some_and(|name| name == upper))
}

pub fn inventory() -> Vec<(u8, &'static str)> {
    (0..=u8::MAX)
        .filter_map(|op| name(op).map(|name| (op, name)))
        .collect()
}
