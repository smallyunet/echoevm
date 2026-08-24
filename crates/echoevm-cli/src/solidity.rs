mod abi;
mod compiler;
mod source_map;

use abi::*;
use compiler::*;
use source_map::*;

use crate::explain::{ExplainOutputArgs, write_explanation};

use alloy_dyn_abi::{JsonAbiExt, Specifier};
use alloy_json_abi::{Function, JsonAbi, Param};
use anyhow::{Context, Result, bail};
use clap::{Args, Subcommand};
use echoevm_core::{Fork, build_evidence, decode_hex, deploy_and_call, explain_evidence};
use echoevm_protocol::ExecutionResult;
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

#[derive(Args, Debug)]
pub struct SolidityExplainArgs {
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
    #[command(flatten)]
    output: ExplainOutputArgs,
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

struct PreparedRun {
    source: PathBuf,
    contract: String,
    function: String,
    compiler_executable: String,
    compiler_version: String,
    duration_ms: u128,
    execution: ExecutionResult,
    source_map: Value,
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
    let prepared = prepare_run(
        &args.compiler,
        args.contract.as_deref(),
        &args.function,
        &args.args,
        &args.constructor_args,
        args.gas,
        args.trace || args.format == "evidence-json",
    )?;
    let base = json!({
        "schemaVersion": 1,
        "source": prepared.source,
        "contract": prepared.contract,
        "function": prepared.function,
        "compiler": { "executable": prepared.compiler_executable, "version": prepared.compiler_version },
        "durationMs": prepared.duration_ms,
        "execution": prepared.execution,
        "sourceMap": prepared.source_map
    });
    match args.format.as_str() {
        "json" => println!("{}", serde_json::to_string_pretty(&base)?),
        "summary-json" => println!(
            "{}",
            serde_json::to_string_pretty(&json!({
                "schemaVersion": 1,
                "source": prepared.source,
                "contract": prepared.contract,
                "function": prepared.function,
                "execution": {
                    "status": prepared.execution.status,
                    "gasUsed": prepared.execution.gas_used,
                    "returnData": prepared.execution.return_data
                }
            }))?
        ),
        "evidence-json" => println!(
            "{}",
            serde_json::to_string_pretty(&solidity_evidence(&prepared, &args.profile, args.limit))?
        ),
        "text" => println!(
            "status={:?} return={} gas={} storage={}",
            prepared.execution.status,
            prepared.execution.return_data,
            prepared.execution.gas_used,
            prepared.execution.storage.len()
        ),
        other => bail!("unsupported Solidity output format {other:?}"),
    }
    Ok(())
}

pub fn explain(args: SolidityExplainArgs) -> Result<()> {
    let prepared = prepare_run(
        &args.compiler,
        args.contract.as_deref(),
        &args.function,
        &args.args,
        &args.constructor_args,
        args.gas,
        true,
    )?;
    let evidence = solidity_evidence(
        &prepared,
        args.output.effective_profile(),
        args.output.limit,
    );
    let input = json!({
        "kind": "solidity-function",
        "source": prepared.source,
        "contract": prepared.contract,
        "function": prepared.function,
        "compiler": {
            "executable": prepared.compiler_executable,
            "version": prepared.compiler_version
        },
        "fork": "Osaka",
        "gasLimit": args.gas,
        "arguments": args.args,
        "constructorArguments": args.constructor_args,
        "runtime": {"name": "EchoEVM", "version": env!("CARGO_PKG_VERSION")}
    });
    let document = explain_evidence(&evidence, input, args.output.expectation()?);
    write_explanation(&document, &args.output.format)
}

fn prepare_run(
    compiler: &CompilerArgs,
    contract_name: Option<&str>,
    function_name: &str,
    arguments: &str,
    constructor_arguments: &str,
    gas: u64,
    include_trace: bool,
) -> Result<PreparedRun> {
    let started = Instant::now();
    let (contracts, version) = compile(compiler)?;
    let contract = select_contract(&contracts, contract_name)?;
    let function = select_function(&contract.abi, function_name)?;
    let mut initcode = decode_hex(&contract.initcode)?;
    if let Some(constructor) = contract.abi.constructor() {
        initcode.extend(
            constructor.abi_encode_input(&coerce(&constructor.inputs, constructor_arguments)?)?,
        );
    } else if !constructor_arguments.trim().is_empty() {
        bail!("contract {} has no constructor arguments", contract.name);
    }
    let calldata = function.abi_encode_input(&coerce(&function.inputs, arguments)?)?;
    let function = function.signature();
    let execution = deploy_and_call(initcode, calldata, gas, Fork::Osaka, include_trace)?;
    Ok(PreparedRun {
        source: compiler.source.clone(),
        contract: contract.key.clone(),
        function,
        compiler_executable: compiler.solc.clone(),
        compiler_version: version,
        duration_ms: started.elapsed().as_millis(),
        execution,
        source_map: runtime_source_map(contract),
    })
}

fn solidity_evidence(prepared: &PreparedRun, profile: &str, limit: usize) -> Value {
    let mut evidence = build_evidence(&prepared.execution, profile, limit);
    if let Some(object) = evidence.as_object_mut() {
        object.insert("source".into(), json!(prepared.source));
        object.insert("contract".into(), json!(prepared.contract));
        object.insert("function".into(), json!(prepared.function));
        object.insert(
            "compiler".into(),
            json!({
                "executable": prepared.compiler_executable,
                "version": prepared.compiler_version
            }),
        );
        if let (Some(events), Some(locations)) = (
            object.get_mut("events").and_then(Value::as_array_mut),
            prepared
                .source_map
                .get("locations")
                .and_then(Value::as_array),
        ) {
            for event in events {
                let Some(pc) = event.get("pc").and_then(Value::as_u64) else {
                    continue;
                };
                if let Some(location) = locations
                    .iter()
                    .find(|location| location.get("pc").and_then(Value::as_u64) == Some(pc))
                    && let Some(event) = event.as_object_mut()
                {
                    event.insert("source".into(), location.clone());
                }
            }
        }
    }
    evidence
}
