mod abi;
mod compiler;
mod source_map;

use abi::*;
use compiler::*;
use source_map::*;

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
                if let (Some(events), Some(locations)) = (
                    object.get_mut("events").and_then(Value::as_array_mut),
                    source_map.get("locations").and_then(Value::as_array),
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
