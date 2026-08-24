use super::*;

pub(super) fn parameters(params: &[Param]) -> Vec<ParameterOutput> {
    params
        .iter()
        .map(|param| ParameterOutput {
            name: param.name.clone(),
            ty: param.selector_type().into_owned(),
        })
        .collect()
}

pub(super) fn function_locations(
    ast: &Value,
    contract_name: &str,
    names: &BTreeMap<i64, String>,
) -> BTreeMap<String, SourceLocation> {
    let mut output = BTreeMap::new();
    let Some(contracts) = ast.get("nodes").and_then(Value::as_array) else {
        return output;
    };
    for contract in contracts {
        if contract.get("nodeType").and_then(Value::as_str) != Some("ContractDefinition")
            || contract.get("name").and_then(Value::as_str) != Some(contract_name)
        {
            continue;
        }
        for function in contract
            .get("nodes")
            .and_then(Value::as_array)
            .into_iter()
            .flatten()
        {
            if function.get("nodeType").and_then(Value::as_str) != Some("FunctionDefinition") {
                continue;
            }
            let Some(selector) = function.get("functionSelector").and_then(Value::as_str) else {
                continue;
            };
            if let Some(location) = parse_source_location(
                function.get("src").and_then(Value::as_str).unwrap_or(""),
                names,
            ) {
                output.insert(selector.to_lowercase(), location);
            }
        }
    }
    output
}

pub(super) fn parse_source_location(
    src: &str,
    names: &BTreeMap<i64, String>,
) -> Option<SourceLocation> {
    let mut fields = src.split(':');
    let start = fields.next()?.parse().ok()?;
    let length = fields.next()?.parse().ok()?;
    let file = names.get(&fields.next()?.parse().ok()?)?.clone();
    Some(SourceLocation {
        file,
        start,
        length,
    })
}

pub(super) fn runtime_source_map(contract: &CompiledContract) -> Value {
    let Ok(code) = decode_hex(&contract.runtime) else {
        return json!({"locations": []});
    };
    let pcs = instruction_pcs(&code);
    let mut start: i64 = -1;
    let mut length: i64 = -1;
    let mut file_id: i64 = -1;
    let mut locations = Vec::new();
    for (index, segment) in contract.source_map.split(';').enumerate() {
        if index >= pcs.len() {
            break;
        }
        let fields: Vec<_> = segment.split(':').collect();
        if !fields.first().unwrap_or(&"").is_empty() {
            start = fields[0].parse().unwrap_or(-1);
        }
        if fields.get(1).is_some_and(|v| !v.is_empty()) {
            length = fields[1].parse().unwrap_or(-1);
        }
        if fields.get(2).is_some_and(|v| !v.is_empty()) {
            file_id = fields[2].parse().unwrap_or(-1);
        }
        if start >= 0
            && length >= 0
            && let Some(file) = contract.source_names.get(&file_id)
        {
            locations
                .push(json!({"pc": pcs[index], "file": file, "start": start, "length": length}));
        }
    }
    json!({"locations": locations})
}

pub(crate) fn instruction_pcs(code: &[u8]) -> Vec<usize> {
    let mut pcs = Vec::new();
    let mut pc = 0;
    while pc < code.len() {
        pcs.push(pc);
        let opcode = code[pc];
        pc += 1 + if (0x60..=0x7f).contains(&opcode) {
            usize::from(opcode - 0x5f)
        } else {
            0
        };
    }
    pcs
}
