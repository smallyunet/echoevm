use alloy_dyn_abi::{JsonAbiExt, Specifier};
use alloy_json_abi::{Function, JsonAbi, Param};
use anyhow::{Context, Result, bail};
use clap::{Args, Subcommand};
use echoevm_core::{Fork, build_evidence, decode_hex, deploy_and_call};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::BTreeMap,
    fs,
    io::Write,
    path::{Path, PathBuf},
    process::{Command, Stdio},
    time::Instant,
};

#[derive(Subcommand, Debug)]
pub enum SolidityCommand {
    Inspect(InspectArgs),
    Run(SolidityRunArgs),
}

#[derive(Args, Debug, Clone)]
pub struct CompilerArgs {
    pub source: PathBuf,
    #[arg(long, default_value = "solc")]
    solc: String,
    #[arg(long = "solc-arg")]
    solc_args: Vec<String>,
    #[arg(long)]
    base_path: Option<PathBuf>,
    #[arg(long = "include-path", value_delimiter = ',')]
    include_paths: Vec<PathBuf>,
    #[arg(long = "remapping")]
    remappings: Vec<String>,
    #[arg(long)]
    optimize: bool,
    #[arg(long, default_value_t = 0)]
    optimizer_runs: u64,
    #[arg(long)]
    via_ir: bool,
}

#[derive(Args, Debug)]
pub struct InspectArgs {
    #[command(flatten)]
    compiler: CompilerArgs,
    #[arg(long, default_value = "json")]
    format: String,
}

#[derive(Args, Debug)]
pub struct SolidityRunArgs {
    #[command(flatten)]
    compiler: CompilerArgs,
    #[arg(long)]
    contract: Option<String>,
    #[arg(long, default_value = "run")]
    function: String,
    #[arg(long, default_value = "")]
    args: String,
    #[arg(long, default_value = "")]
    constructor_args: String,
    #[arg(long, default_value_t = 1_000_000)]
    gas: u64,
    #[arg(long, default_value = "json")]
    format: String,
    #[arg(long)]
    trace: bool,
    #[arg(long, default_value = "auto")]
    profile: String,
    #[arg(long, default_value_t = 40)]
    limit: usize,
}

#[derive(Clone, Debug)]
struct CompiledContract {
    key: String,
    name: String,
    initcode: String,
    runtime: String,
    source_map: String,
    abi: JsonAbi,
    source_names: BTreeMap<i64, String>,
    function_locations: BTreeMap<String, SourceLocation>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct SourceLocation {
    file: String,
    start: usize,
    length: usize,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct ParameterOutput {
    #[serde(skip_serializing_if = "String::is_empty")]
    name: String,
    #[serde(rename = "type")]
    ty: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct FunctionOutput {
    name: String,
    signature: String,
    inputs: Vec<ParameterOutput>,
    outputs: Vec<ParameterOutput>,
    state_mutability: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    source_location: Option<SourceLocation>,
}

pub fn execute(command: SolidityCommand) -> Result<()> {
    match command {
        SolidityCommand::Inspect(args) => inspect(args),
        SolidityCommand::Run(args) => run(args),
    }
}

fn inspect(args: InspectArgs) -> Result<()> {
    let started = Instant::now();
    let (contracts, version) = compile(&args.compiler)?;
    let items: Vec<Value> = contracts
        .iter()
        .map(|contract| {
            let mut functions: Vec<_> = contract
                .abi
                .functions()
                .map(|function| FunctionOutput {
                    name: function.name.clone(),
                    signature: function.signature(),
                    inputs: parameters(&function.inputs),
                    outputs: parameters(&function.outputs),
                    state_mutability: function.state_mutability.as_json_str().into(),
                    source_location: contract
                        .function_locations
                        .get(&hex::encode(function.selector()))
                        .cloned(),
                })
                .collect();
            functions.sort_by(|a, b| a.signature.cmp(&b.signature));
            json!({
                "key": contract.key,
                "name": contract.name,
                "constructorInputs": contract.abi.constructor().map(|c| parameters(&c.inputs)).unwrap_or_default(),
                "functions": functions
            })
        })
        .collect();
    let output = json!({
        "schemaVersion": 1,
        "source": args.compiler.source,
        "compiler": { "executable": args.compiler.solc, "version": version },
        "durationMs": started.elapsed().as_millis(),
        "contracts": items
    });
    if args.format == "json" {
        println!("{}", serde_json::to_string_pretty(&output)?);
    } else if args.format == "text" {
        for contract in contracts {
            println!("{} ({})", contract.name, contract.key);
            for function in contract.abi.functions() {
                println!("  {}", function.signature());
            }
        }
    } else {
        bail!("unsupported inspect format {:?}", args.format);
    }
    Ok(())
}

fn run(args: SolidityRunArgs) -> Result<()> {
    let started = Instant::now();
    let (contracts, version) = compile(&args.compiler)?;
    let contract = select_contract(&contracts, args.contract.as_deref())?;
    let function = select_function(&contract.abi, &args.function)?;
    let mut initcode = decode_hex(&contract.initcode)?;
    if let Some(constructor) = contract.abi.constructor() {
        initcode.extend(
            constructor.abi_encode_input(&coerce(&constructor.inputs, &args.constructor_args)?)?,
        );
    } else if !args.constructor_args.trim().is_empty() {
        bail!("contract {} has no constructor arguments", contract.name);
    }
    let calldata = function.abi_encode_input(&coerce(&function.inputs, &args.args)?)?;
    let include_trace = args.trace || args.format == "evidence-json";
    let execution = deploy_and_call(initcode, calldata, args.gas, Fork::Osaka, include_trace)?;
    let source_map = runtime_source_map(contract);
    let base = json!({
        "schemaVersion": 1,
        "source": args.compiler.source,
        "contract": contract.key,
        "function": function.signature(),
        "compiler": { "executable": args.compiler.solc, "version": version },
        "durationMs": started.elapsed().as_millis(),
        "execution": execution,
        "sourceMap": source_map
    });
    match args.format.as_str() {
        "json" => println!("{}", serde_json::to_string_pretty(&base)?),
        "summary-json" => println!(
            "{}",
            serde_json::to_string_pretty(&json!({
                "schemaVersion": 1,
                "source": args.compiler.source,
                "contract": contract.key,
                "function": function.signature(),
                "execution": { "status": execution.status, "gasUsed": execution.gas_used, "returnData": execution.return_data }
            }))?
        ),
        "evidence-json" => {
            let mut evidence = build_evidence(&execution, &args.profile, args.limit);
            if let Some(object) = evidence.as_object_mut() {
                object.insert("source".into(), json!(args.compiler.source));
                object.insert("contract".into(), json!(contract.key));
                object.insert("function".into(), json!(function.signature()));
                object.insert(
                    "compiler".into(),
                    json!({"executable": args.compiler.solc, "version": version}),
                );
            }
            println!("{}", serde_json::to_string_pretty(&evidence)?);
        }
        "text" => println!(
            "status={:?} return={} gas={} storage={}",
            execution.status,
            execution.return_data,
            execution.gas_used,
            execution.storage.len()
        ),
        other => bail!("unsupported Solidity output format {other:?}"),
    }
    Ok(())
}

fn compile(args: &CompilerArgs) -> Result<(Vec<CompiledContract>, String)> {
    let source = fs::canonicalize(&args.source)
        .with_context(|| format!("read Solidity source {}", args.source.display()))?;
    let base = fs::canonicalize(
        args.base_path
            .clone()
            .unwrap_or_else(|| source.parent().unwrap_or(Path::new(".")).to_owned()),
    )?;
    let key = source
        .strip_prefix(&base)
        .unwrap_or(&source)
        .to_string_lossy()
        .replace('\\', "/");
    let content = fs::read_to_string(&source)?;
    let input = json!({
        "language": "Solidity",
        "sources": { key.clone(): { "content": content } },
        "settings": {
            "optimizer": { "enabled": args.optimize, "runs": args.optimizer_runs },
            "viaIR": args.via_ir,
            "remappings": args.remappings,
            "evmVersion": "cancun",
            "outputSelection": { "*": { "*": ["abi", "evm.bytecode.object", "evm.deployedBytecode.object", "evm.deployedBytecode.sourceMap"], "": ["ast"] } }
        }
    });
    let mut command = Command::new(&args.solc);
    command
        .args(&args.solc_args)
        .arg("--standard-json")
        .arg("--base-path")
        .arg(&base);
    for path in &args.include_paths {
        command.arg("--include-path").arg(fs::canonicalize(path)?);
    }
    let mut child = command
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .with_context(|| format!("start Solidity compiler {}", args.solc))?;
    child
        .stdin
        .take()
        .expect("piped stdin")
        .write_all(&serde_json::to_vec(&input)?)?;
    let output = child.wait_with_output()?;
    if !output.status.success() {
        bail!(
            "solc compilation failed: {}",
            String::from_utf8_lossy(&output.stderr).trim()
        );
    }
    let compiled: SolcOutput =
        serde_json::from_slice(&output.stdout).context("parse solc standard JSON output")?;
    let errors: Vec<_> = compiled
        .errors
        .iter()
        .filter(|error| error.severity == "error")
        .map(|error| error.formatted_message.as_deref().unwrap_or(&error.message))
        .collect();
    if !errors.is_empty() {
        bail!("solc compilation failed: {}", errors.join("\n"));
    }
    let source_names: BTreeMap<_, _> = compiled
        .sources
        .iter()
        .map(|(name, source)| (source.id, name.clone()))
        .collect();
    let mut contracts = Vec::new();
    for (source_name, by_name) in compiled.contracts {
        for (name, artifact) in by_name {
            if artifact.evm.deployed_bytecode.object.is_empty() {
                continue;
            }
            let abi: JsonAbi = serde_json::from_value(artifact.abi)?;
            let locations = compiled
                .sources
                .get(&source_name)
                .map(|source| function_locations(&source.ast, &name, &source_names))
                .unwrap_or_default();
            contracts.push(CompiledContract {
                key: format!("{source_name}:{name}"),
                name,
                initcode: artifact.evm.bytecode.object,
                runtime: artifact.evm.deployed_bytecode.object,
                source_map: artifact.evm.deployed_bytecode.source_map,
                abi,
                source_names: source_names.clone(),
                function_locations: locations,
            });
        }
    }
    contracts.sort_by(|a, b| a.key.cmp(&b.key));
    if contracts.is_empty() {
        bail!("solc produced no deployable contracts");
    }
    Ok((contracts, compiler_version(args)))
}

fn compiler_version(args: &CompilerArgs) -> String {
    Command::new(&args.solc)
        .args(&args.solc_args)
        .arg("--version")
        .output()
        .ok()
        .map(|output| {
            String::from_utf8_lossy(&output.stdout)
                .lines()
                .last()
                .unwrap_or("unknown")
                .trim()
                .trim_start_matches("Version:")
                .trim()
                .to_owned()
        })
        .filter(|value| !value.is_empty())
        .unwrap_or_else(|| "unknown".into())
}

fn select_contract<'a>(
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

fn select_function<'a>(abi: &'a JsonAbi, requested: &str) -> Result<&'a Function> {
    let matches: Vec<_> = abi
        .functions()
        .filter(|function| function.name == requested || function.signature() == requested)
        .collect();
    if matches.len() == 1 {
        return Ok(matches[0]);
    }
    bail!("function {requested:?} did not uniquely identify an ABI function")
}

fn coerce(params: &[Param], input: &str) -> Result<Vec<alloy_dyn_abi::DynSolValue>> {
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

fn split_args(input: &str) -> Vec<&str> {
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

fn parameters(params: &[Param]) -> Vec<ParameterOutput> {
    params
        .iter()
        .map(|param| ParameterOutput {
            name: param.name.clone(),
            ty: param.selector_type().into_owned(),
        })
        .collect()
}

fn function_locations(
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

fn parse_source_location(src: &str, names: &BTreeMap<i64, String>) -> Option<SourceLocation> {
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

fn runtime_source_map(contract: &CompiledContract) -> Value {
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

fn instruction_pcs(code: &[u8]) -> Vec<usize> {
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

#[derive(Deserialize)]
struct SolcOutput {
    #[serde(default)]
    contracts: BTreeMap<String, BTreeMap<String, SolcContract>>,
    #[serde(default)]
    sources: BTreeMap<String, SolcSource>,
    #[serde(default)]
    errors: Vec<SolcError>,
}

#[derive(Deserialize)]
struct SolcContract {
    abi: Value,
    evm: SolcEvm,
}
#[derive(Deserialize)]
struct SolcEvm {
    bytecode: SolcBytecode,
    #[serde(rename = "deployedBytecode")]
    deployed_bytecode: SolcDeployedBytecode,
}
#[derive(Deserialize)]
struct SolcBytecode {
    #[serde(default)]
    object: String,
}
#[derive(Deserialize)]
struct SolcDeployedBytecode {
    #[serde(default)]
    object: String,
    #[serde(default, rename = "sourceMap")]
    source_map: String,
}
#[derive(Deserialize)]
struct SolcSource {
    id: i64,
    #[serde(default)]
    ast: Value,
}
#[derive(Deserialize)]
struct SolcError {
    severity: String,
    message: String,
    #[serde(rename = "formattedMessage")]
    formatted_message: Option<String>,
}
