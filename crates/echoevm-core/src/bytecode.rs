use super::*;

pub fn disassemble(bytecode: &[u8]) -> Vec<String> {
    let mut pc = 0usize;
    let mut lines = Vec::new();
    while pc < bytecode.len() {
        let opcode = bytecode[pc];
        let name = opcode::name(opcode).unwrap_or("UNKNOWN");
        let width = if (0x60..=0x7f).contains(&opcode) {
            usize::from(opcode - 0x5f)
        } else {
            0
        };
        let end = (pc + 1 + width).min(bytecode.len());
        let argument = if width == 0 {
            String::new()
        } else {
            format!(" 0x{}", hex::encode(&bytecode[pc + 1..end]))
        };
        lines.push(format!("{pc:04x}: {name}{argument}"));
        pc = end;
    }
    lines
}

pub fn assemble(input: &str) -> Result<Vec<u8>, ExecuteError> {
    if !input.contains(char::is_whitespace) {
        return decode_hex(input);
    }
    let mut output = Vec::new();
    for token in input.split_whitespace() {
        let upper = token.to_ascii_uppercase();
        if let Some(opcode) = opcode::by_name(&upper) {
            output.push(opcode);
        } else {
            output.extend(decode_hex(token)?);
        }
    }
    Ok(output)
}
