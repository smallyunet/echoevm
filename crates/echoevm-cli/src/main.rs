use anyhow::{Context, Result, bail};
use clap::{Args, Parser, Subcommand, ValueEnum};
use echoevm_core::{
    ExecuteRequest, Fork, build_evidence, decode_hex, deploy_bytecode, disassemble, execute,
    replay_witness, trace,
};
use std::{fs, path::PathBuf};

mod explain;
mod foundry;
mod repl;
mod solidity;
mod web;
mod witness;

#[derive(Parser, Debug)]
#[command(
    name = "echoevm",
    version,
    about = "Independent Rust EVM execution evidence"
)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand, Debug)]
enum Command {
    Run(RunArgs),
    Trace(TraceArgs),
    Disasm(CodeArgs),
    Version(VersionArgs),
    Replay(ReplayArgs),
    Explain {
        #[command(subcommand)]
        command: explain::ExplainCommand,
    },
    Call(RunArgs),
    Deploy(RunArgs),
    Solidity {
        #[command(subcommand)]
        command: solidity::SolidityCommand,
    },
    Repl,
    Web(WebArgs),
    Witness {
        #[command(subcommand)]
        command: WitnessCommand,
    },
}

#[derive(Args, Debug)]
struct WebArgs {
    #[arg(long, default_value = "127.0.0.1:8080")]
    addr: String,
    #[arg(long)]
    code: Option<String>,
}

#[derive(Subcommand, Debug)]
enum WitnessCommand {
    /// Build a proof-verified witness using standard JSON-RPC methods.
    ImportProof {
        input: String,
        #[arg(long, env = "ETHEREUM_RPC_URL")]
        rpc_url: String,
        #[arg(long)]
        out: Option<PathBuf>,
        /// Optional durable copy of the EIP-1186 proof material.
        #[arg(long)]
        proofs_out: Option<PathBuf>,
        /// Number of historical block hashes to embed for BLOCKHASH execution.
        #[arg(long, default_value_t = 256, value_parser = clap::value_parser!(u16).range(0..=256))]
        blockhash_depth: u16,
    },
    ImportDebug {
        input: String,
        #[arg(long, env = "ETHEREUM_RPC_URL")]
        rpc_url: String,
        #[arg(long)]
        out: Option<PathBuf>,
    },
    /// Export one bounded runtime call from a Foundry JSON artifact.
    FromFoundry(foundry::FoundryWitnessArgs),
}

#[derive(Clone, Copy, Debug, Default, ValueEnum)]
enum ForkArg {
    Cancun,
    Prague,
    #[default]
    Osaka,
}

impl From<ForkArg> for Fork {
    fn from(value: ForkArg) -> Self {
        match value {
            ForkArg::Cancun => Fork::Cancun,
            ForkArg::Prague => Fork::Prague,
            ForkArg::Osaka => Fork::Osaka,
        }
    }
}

#[derive(Args, Debug)]
struct CodeArgs {
    bytecode: Option<String>,
    #[arg(long)]
    path: Option<PathBuf>,
}

#[derive(Args, Debug)]
struct RunArgs {
    bytecode: Option<String>,
    #[arg(long, short = 'r')]
    bin_runtime: Option<PathBuf>,
    #[arg(long, short = 'd', default_value = "")]
    calldata: String,
    #[arg(long, default_value_t = echoevm_core::DEFAULT_GAS_LIMIT)]
    gas: u64,
    #[arg(long, value_enum, default_value_t)]
    fork: ForkArg,
    #[arg(long)]
    json: bool,
    #[arg(long)]
    debug: bool,
}

#[derive(Args, Debug)]
struct TraceArgs {
    #[command(flatten)]
    run: RunArgs,
    #[arg(long, default_value = "jsonl")]
    format: String,
    #[arg(long, default_value_t = 0)]
    limit: usize,
    #[arg(long, default_value = "auto")]
    profile: String,
}

#[derive(Args, Debug)]
struct VersionArgs {
    #[arg(long)]
    json: bool,
}

#[derive(Args, Debug)]
struct ReplayArgs {
    witness: PathBuf,
    #[arg(long, default_value = "text")]
    format: String,
    #[arg(long, default_value = "auto")]
    profile: String,
    #[arg(long, default_value_t = 40)]
    limit: usize,
}

fn read_code(inline: Option<&str>, path: Option<&PathBuf>) -> Result<Vec<u8>> {
    let value = if let Some(path) = path {
        fs::read_to_string(path).with_context(|| format!("read {}", path.display()))?
    } else if let Some(value) = inline {
        value.to_owned()
    } else {
        bail!("bytecode or --bin-runtime is required")
    };
    decode_hex(&value).map_err(Into::into)
}

fn run(args: &RunArgs) -> Result<()> {
    let bytecode = read_code(args.bytecode.as_deref(), args.bin_runtime.as_ref())?;
    let result = execute(ExecuteRequest {
        bytecode,
        calldata: decode_hex(&args.calldata)?,
        gas_limit: args.gas,
        fork: args.fork.into(),
    })?;
    if args.json {
        println!("{}", serde_json::to_string_pretty(&result)?);
    } else {
        println!("status   {:?}", result.status);
        println!("gas used {}", result.gas_used);
        println!("return   {}", result.return_data);
        if let Some(error) = result.error {
            println!("error    {error}");
        }
    }
    Ok(())
}

fn deploy(args: &RunArgs) -> Result<()> {
    let initcode = read_code(args.bytecode.as_deref(), args.bin_runtime.as_ref())?;
    let result = deploy_bytecode(initcode, args.gas, args.fork.into(), args.debug)?;
    if args.json {
        println!("{}", serde_json::to_string_pretty(&result)?);
    } else {
        println!("status   {:?}", result.status);
        println!("gas used {}", result.gas_used);
        println!("runtime  {}", result.return_data);
    }
    Ok(())
}

fn trace_code(args: &TraceArgs) -> Result<()> {
    let bytecode = read_code(args.run.bytecode.as_deref(), args.run.bin_runtime.as_ref())?;
    let mut result = trace(ExecuteRequest {
        bytecode,
        calldata: decode_hex(&args.run.calldata)?,
        gas_limit: args.run.gas,
        fork: args.run.fork.into(),
    })?;
    let full_result = result.clone();
    if args.limit > 0
        && let Some(steps) = &mut result.trace
    {
        steps.truncate(args.limit);
    }
    match args.format.as_str() {
        "json" => println!("{}", serde_json::to_string_pretty(&result)?),
        "jsonl" => {
            for step in result.trace.unwrap_or_default() {
                println!("{}", serde_json::to_string(&step)?);
            }
        }
        "text" => {
            for step in result.trace.unwrap_or_default() {
                println!(
                    "{:04} pc={:04x} op={:<14} gas={} stack={:?}",
                    step.index, step.pc, step.opcode_name, step.gas_before, step.stack_before
                );
            }
        }
        "evidence-json" => println!(
            "{}",
            serde_json::to_string_pretty(&build_evidence(&full_result, &args.profile, args.limit))?
        ),
        other => {
            bail!("unsupported trace format {other:?}; use text, json, jsonl, or evidence-json")
        }
    }
    Ok(())
}

fn main() -> Result<()> {
    let cli = Cli::parse();
    match cli.command {
        Command::Run(args) | Command::Call(args) => run(&args),
        Command::Deploy(args) => deploy(&args),
        Command::Trace(args) => trace_code(&args),
        Command::Disasm(args) => {
            for line in disassemble(&read_code(args.bytecode.as_deref(), args.path.as_ref())?) {
                println!("{line}");
            }
            Ok(())
        }
        Command::Version(args) => {
            if args.json {
                println!(
                    "{}",
                    serde_json::json!({"version": env!("CARGO_PKG_VERSION"), "runtime": "rust", "rustVersion": option_env!("RUSTC_VERSION").unwrap_or("unknown")})
                );
            } else {
                println!("echoevm v{} rust", env!("CARGO_PKG_VERSION"));
            }
            Ok(())
        }
        Command::Replay(args) => {
            let bytes = fs::read(&args.witness)?;
            let witness = echoevm_protocol::ReplayWitness::decode_strict(&bytes)?;
            let mut result = replay_witness(&witness, true)?;
            result.evidence = Some(build_evidence(&result.execution, &args.profile, args.limit));
            match args.format.as_str() {
                "json" => {
                    println!("{}", serde_json::to_string_pretty(&result)?)
                }
                "evidence-json" => {
                    println!("{}", serde_json::to_string_pretty(&result.evidence)?)
                }
                "text" => {
                    println!("transaction {}", result.transaction.hash);
                    println!("status      {}", result.transaction.status);
                    println!("gas used    {}", result.execution.gas_used);
                    println!(
                        "trace steps {}",
                        result.execution.trace.as_ref().map_or(0, Vec::len)
                    );
                    println!("witness     sha256={}", result.witness.sha256);
                }
                other => {
                    bail!("unsupported replay format {other:?}; use text, json, or evidence-json")
                }
            }
            Ok(())
        }
        Command::Explain { command } => explain::execute(command),
        Command::Solidity { command } => solidity::execute(command),
        Command::Repl => repl::run(),
        Command::Web(args) => web::run(&args.addr, args.code.as_deref()),
        Command::Witness { command } => match command {
            WitnessCommand::ImportProof {
                input,
                rpc_url,
                out,
                proofs_out,
                blockhash_depth,
            } => witness::import_proof(
                &input,
                &rpc_url,
                out.as_deref(),
                proofs_out.as_deref(),
                blockhash_depth,
            ),
            WitnessCommand::ImportDebug {
                input,
                rpc_url,
                out,
            } => witness::import_debug(&input, &rpc_url, out.as_deref()),
            WitnessCommand::FromFoundry(args) => foundry::export(args),
        },
    }
}
