use crate::solidity;
use anyhow::{Result, bail};
use clap::{Args, Subcommand, ValueEnum};
use echoevm_core::{
    build_evidence, decode_hex, execute_test_witness, explain_evidence, replay_witness,
};
use echoevm_protocol::{
    ExecutionStatus, ExplanationDocument, ExplanationExpectation, ReplayWitness, TestWitness,
};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::{fs, path::PathBuf};

#[derive(Subcommand, Debug)]
pub enum ExplainCommand {
    /// Explain a self-contained transaction witness without contacting RPC.
    Replay(ExplainReplayArgs),
    /// Explain a self-contained call-level test witness.
    Test(ExplainTestArgs),
    /// Compile and explain one Solidity function execution.
    Solidity(Box<solidity::SolidityExplainArgs>),
    /// Prepare and explain one self-contained Foundry test function.
    Foundry(Box<crate::foundry::FoundryExplainArgs>),
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
            storage: Default::default(),
        })
    }
}

#[derive(Args, Debug)]
pub struct ExplainReplayArgs {
    pub witness: PathBuf,
    #[command(flatten)]
    pub output: ExplainOutputArgs,
}

#[derive(Args, Debug)]
pub struct ExplainTestArgs {
    pub witness: PathBuf,
    #[arg(long, default_value = "text", value_parser = ["text", "json"])]
    pub format: String,
    #[arg(long, default_value = "auto")]
    pub profile: String,
    #[arg(long, default_value_t = 40)]
    pub limit: usize,
}

pub fn execute(command: ExplainCommand) -> Result<()> {
    match command {
        ExplainCommand::Replay(args) => replay(args),
        ExplainCommand::Test(args) => test(args),
        ExplainCommand::Solidity(args) => solidity::explain(*args),
        ExplainCommand::Foundry(args) => crate::foundry::explain(*args),
    }
}

fn test(args: ExplainTestArgs) -> Result<()> {
    let bytes = fs::read(&args.witness)?;
    let witness = TestWitness::decode_strict(&bytes)?;
    let input = json!({
        "kind": "test-witness",
        "path": args.witness,
        "schema": witness.schema,
        "name": witness.name,
        "sha256": hex::encode(Sha256::digest(&bytes)),
        "fork": format!("{:?}", witness.fork),
        "gasLimit": witness.gas_limit,
        "calldata": format!("0x{}", hex::encode(&witness.calldata)),
        "source": witness.source,
        "runtime": {"name": "EchoEVM", "version": env!("CARGO_PKG_VERSION")}
    });
    let document = explain_test_witness(&witness, input, &args.profile, args.limit)?;
    write_explanation(&document, &args.format)
}

pub(crate) fn explain_test_witness(
    witness: &TestWitness,
    input: Value,
    requested_profile: &str,
    limit: usize,
) -> Result<ExplanationDocument> {
    let execution = execute_test_witness(witness, true)?;
    let profile = test_profile(requested_profile, witness);
    let mut evidence = build_evidence(&execution, profile, limit);
    enrich_test_sources(&mut evidence, witness);
    let expectation = ExplanationExpectation {
        status: witness.expectation.status.as_ref().map(execution_status),
        return_data: witness
            .expectation
            .return_data
            .as_ref()
            .map(|value| format!("0x{}", hex::encode(value))),
        storage: witness
            .expectation
            .storage
            .iter()
            .map(|(slot, value)| (slot.to_string(), value.to_string()))
            .collect(),
    };
    Ok(explain_evidence(&evidence, input, expectation))
}

fn test_profile<'a>(requested: &'a str, witness: &TestWitness) -> &'a str {
    if requested != "auto" {
        return requested;
    }
    let has_return = witness.expectation.return_data.is_some();
    let has_storage = !witness.expectation.storage.is_empty();
    match (
        has_return,
        has_storage,
        witness.expectation.status.is_some(),
    ) {
        (true, true, _) => "full",
        (true, false, _) => "arithmetic",
        (false, true, _) => "storage",
        (false, false, true) => "revert",
        (false, false, false) => "auto",
    }
}

fn execution_status(status: &ExecutionStatus) -> String {
    match status {
        ExecutionStatus::Success => "success",
        ExecutionStatus::Revert => "revert",
        ExecutionStatus::Fault => "fault",
    }
    .into()
}

fn enrich_test_sources(evidence: &mut Value, witness: &TestWitness) {
    let Some(source) = &witness.source else {
        return;
    };
    let Some(events) = evidence.get_mut("events").and_then(Value::as_array_mut) else {
        return;
    };
    for event in events {
        if event.get("depth").and_then(Value::as_u64) != Some(0) {
            continue;
        }
        let Some(pc) = event.get("pc").and_then(Value::as_u64) else {
            continue;
        };
        if let Some(location) = source.locations.iter().find(|location| location.pc == pc)
            && let Some(event) = event.as_object_mut()
        {
            event.insert("source".into(), json!(location));
        }
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
