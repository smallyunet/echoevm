use crate::solidity;
use anyhow::{Result, bail};
use clap::{Args, Subcommand, ValueEnum};
use echoevm_core::{build_evidence, decode_hex, explain_evidence, replay_witness};
use echoevm_protocol::{ExplanationDocument, ExplanationExpectation, ReplayWitness};
use serde_json::json;
use std::{fs, path::PathBuf};

#[derive(Subcommand, Debug)]
pub enum ExplainCommand {
    /// Explain a self-contained transaction witness without contacting RPC.
    Replay(ExplainReplayArgs),
    /// Compile and explain one Solidity function execution.
    Solidity(Box<solidity::SolidityExplainArgs>),
}

#[derive(Clone, Copy, Debug, ValueEnum)]
pub enum ExpectedStatus {
    Success,
    Revert,
    Fault,
}

impl ExpectedStatus {
    fn as_str(self) -> &'static str {
        match self {
            Self::Success => "success",
            Self::Revert => "revert",
            Self::Fault => "fault",
        }
    }
}

#[derive(Args, Clone, Debug)]
pub struct ExplainOutputArgs {
    /// Human-readable report or the stable echoevm.explanation.v1 document.
    #[arg(long, default_value = "text", value_parser = ["text", "json"])]
    pub format: String,
    #[arg(long, default_value = "auto")]
    pub profile: String,
    #[arg(long, default_value_t = 40)]
    pub limit: usize,
    #[arg(long, value_enum)]
    pub expect_status: Option<ExpectedStatus>,
    /// Expected ABI return data as hexadecimal bytes.
    #[arg(long)]
    pub expect_return: Option<String>,
}

impl ExplainOutputArgs {
    pub fn effective_profile(&self) -> &str {
        if self.profile == "auto" && self.expect_return.is_some() {
            "arithmetic"
        } else {
            &self.profile
        }
    }

    pub fn expectation(&self) -> Result<ExplanationExpectation> {
        let return_data = self
            .expect_return
            .as_deref()
            .map(|value| decode_hex(value).map(|bytes| format!("0x{}", hex::encode(bytes))))
            .transpose()?;
        Ok(ExplanationExpectation {
            status: self.expect_status.map(|status| status.as_str().into()),
            return_data,
        })
    }
}

#[derive(Args, Debug)]
pub struct ExplainReplayArgs {
    pub witness: PathBuf,
    #[command(flatten)]
    pub output: ExplainOutputArgs,
}

pub fn execute(command: ExplainCommand) -> Result<()> {
    match command {
        ExplainCommand::Replay(args) => replay(args),
        ExplainCommand::Solidity(args) => solidity::explain(*args),
    }
}

fn replay(args: ExplainReplayArgs) -> Result<()> {
    let bytes = fs::read(&args.witness)?;
    let witness = ReplayWitness::decode_strict(&bytes)?;
    let result = replay_witness(&witness, true)?;
    let evidence = build_evidence(
        &result.execution,
        args.output.effective_profile(),
        args.output.limit,
    );
    let input = json!({
        "kind": "transaction-witness",
        "path": args.witness,
        "runtime": {"name": "EchoEVM", "version": env!("CARGO_PKG_VERSION")},
        "transaction": result.transaction,
        "witness": result.witness,
        "warnings": result.warnings,
    });
    let document = explain_evidence(&evidence, input, args.output.expectation()?);
    write_explanation(&document, &args.output.format)
}

pub fn write_explanation(document: &ExplanationDocument, format: &str) -> Result<()> {
    match format {
        "json" => println!("{}", serde_json::to_string_pretty(document)?),
        "text" => {
            println!("Verdict    {}", document.verdict.code);
            println!("Summary    {}", document.verdict.summary);
            if let Some(root) = &document.root_cause {
                println!("Root cause {}", root.code);
                println!("Cause      {}", root.summary);
            } else {
                println!("Root cause not established");
            }
            let status = document
                .execution
                .get("status")
                .and_then(serde_json::Value::as_str)
                .unwrap_or("unknown");
            let gas = document
                .execution
                .get("gasUsed")
                .and_then(serde_json::Value::as_u64)
                .unwrap_or_default();
            println!("Execution  status={status} gasUsed={gas}");
            println!(
                "Evidence   selected={} omitted={} truncated={}",
                document.evidence.selected, document.evidence.omitted, document.evidence.truncated
            );
            for finding in &document.findings {
                println!("Finding    {}: {}", finding.code, finding.summary);
                for reference in &finding.evidence {
                    let source = reference
                        .source
                        .as_ref()
                        .and_then(|source| source.get("file"))
                        .and_then(serde_json::Value::as_str)
                        .map(|file| format!(" source={file}"))
                        .unwrap_or_default();
                    println!(
                        "  evidence step={} depth={} pc={} op={}{}",
                        reference.step, reference.depth, reference.pc, reference.op, source
                    );
                }
            }
            for limitation in &document.limitations {
                println!("Limitation {limitation}");
            }
        }
        other => bail!("unsupported explain format {other:?}; use text or json"),
    }
    Ok(())
}
