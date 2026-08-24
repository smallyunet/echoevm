use super::*;

pub(super) fn select_contract<'a>(
    contracts: &'a [CompiledContract],
    requested: Option<&str>,
) -> Result<&'a CompiledContract> {
    if let Some(requested) = requested {
        let matches: Vec<_> = contracts
            .iter()
            .filter(|c| c.key == requested || c.name == requested)
            .collect();
        if matches.len() == 1 {
            return Ok(matches[0]);
        }
        bail!("contract {requested:?} did not uniquely identify a deployable contract");
    }
    if contracts.len() != 1 {
        bail!("multiple contracts found; use --contract");
    }
    Ok(&contracts[0])
}

pub(super) fn select_function<'a>(abi: &'a JsonAbi, requested: &str) -> Result<&'a Function> {
    let matches: Vec<_> = abi
        .functions()
        .filter(|function| function.name == requested || function.signature() == requested)
        .collect();
    if matches.len() == 1 {
        return Ok(matches[0]);
    }
    bail!("function {requested:?} did not uniquely identify an ABI function")
}

pub(super) fn coerce(params: &[Param], input: &str) -> Result<Vec<alloy_dyn_abi::DynSolValue>> {
    let values = split_args(input);
    if params.len() != values.len() {
        bail!(
            "argument count mismatch: expected {}, got {}",
            params.len(),
            values.len()
        );
    }
    params
        .iter()
        .zip(values)
        .map(|(param, value)| Ok(param.resolve()?.coerce_str(value.trim())?))
        .collect()
}

pub(super) fn split_args(input: &str) -> Vec<&str> {
    if input.trim().is_empty() {
        return Vec::new();
    }
    let mut depth = 0i32;
    let mut quoted = false;
    let mut start = 0;
    let bytes = input.as_bytes();
    let mut result = Vec::new();
    for (index, byte) in bytes.iter().enumerate() {
        match *byte {
            b'"' if index == 0 || bytes[index - 1] != b'\\' => quoted = !quoted,
            b'[' | b'(' if !quoted => depth += 1,
            b']' | b')' if !quoted => depth -= 1,
            b',' if !quoted && depth == 0 => {
                result.push(&input[start..index]);
                start = index + 1;
            }
            _ => {}
        }
    }
    result.push(&input[start..]);
    result
}
