//! Stable EchoEVM v1 wire contracts shared by native and Wasm frontends.

use alloy_primitives::{Address, B256, Bytes, U256};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

pub const TRACE_SCHEMA: &str = "echoevm.trace.v1";
pub const EVIDENCE_SCHEMA: &str = "echoevm.evidence.v1";
pub const WITNESS_SCHEMA: &str = "echoevm.replay-witness.v1";
pub const PROTOCOL_VERSION: u32 = 1;
pub const MAX_WITNESS_BYTES: usize = 64 << 20;

fn bytes_is_empty(value: &Bytes) -> bool {
    value.as_ref().is_empty()
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct TraceStep {
    pub index: usize,
    pub depth: usize,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub address: Option<String>,
    pub pc: u64,
    pub opcode: String,
    pub opcode_name: String,
    pub gas_before: u64,
    pub gas_after: u64,
    #[serde(default)]
    pub stack_before: Vec<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub stack_after: Option<Vec<String>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub halt_class: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "lowercase")]
pub enum ExecutionStatus {
    #[default]
    Success,
    Revert,
    Fault,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct ExecutionResult {
    pub engine: String,
    pub engine_version: String,
    pub status: ExecutionStatus,
    pub return_data: String,
    pub gas_used: u64,
    #[serde(default)]
    pub storage: BTreeMap<String, String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub trace: Option<Vec<TraceStep>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct TransactionSummary {
    pub hash: String,
    pub explorer_url: String,
    pub chain_id: u64,
    pub block_number: u64,
    pub block_hash: String,
    pub transaction_index: u64,
    pub from: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub to: Option<String>,
    pub value: String,
    pub gas_limit: u64,
    pub gas_used: u64,
    #[serde(rename = "type")]
    pub transaction_type: u8,
    pub input: String,
    pub status: String,
    pub fork: String,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
pub struct WitnessProvenance {
    pub schema: String,
    pub sha256: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
pub struct ReplayResult {
    pub transaction: TransactionSummary,
    pub execution: ExecutionResult,
    #[serde(default)]
    pub state: BTreeMap<String, String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub warnings: Vec<String>,
    pub witness: WitnessProvenance,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub evidence: Option<serde_json::Value>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct TraceExecution {
    pub status: String,
    pub gas_limit: u64,
    pub gas_used: u64,
    pub return_data: String,
    pub total_steps: usize,
    pub matched_steps: usize,
    pub emitted_steps: usize,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub filtered: bool,
    pub truncated: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct OpcodeEvent {
    #[serde(rename = "type")]
    pub event_type: String,
    pub schema: String,
    pub step: usize,
    pub depth: usize,
    pub address: String,
    pub pc: u64,
    pub opcode: String,
    pub opcode_name: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub gas: Option<GasDelta>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub stack: Option<StackDelta>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub storage: Vec<StorageAccess>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub control: Option<ControlFlow>,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub halt: bool,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub reverted: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub explanation: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct GasDelta {
    pub before: u64,
    pub after: u64,
    pub used: u64,
    pub static_cost: u64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub dynamic_cost: Option<u64>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct StackDelta {
    pub size_before: usize,
    pub size_after: usize,
    #[serde(default)]
    pub popped: Vec<String>,
    #[serde(default)]
    pub pushed: Vec<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct StorageAccess {
    pub kind: String,
    pub address: String,
    pub slot: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub previous: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub value: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct ControlFlow {
    pub kind: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub target: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub destination: Option<u64>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
pub struct TraceDocument {
    pub schema: String,
    pub execution: TraceExecution,
    pub events: Vec<OpcodeEvent>,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct ReplayWitness {
    pub schema: String,
    pub chain_id: u64,
    pub block_hash: B256,
    pub transaction_index: u64,
    pub header: serde_json::Value,
    pub transaction: Bytes,
    pub prestate: BTreeMap<Address, WitnessAccount>,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub block_hashes: BTreeMap<String, B256>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct WitnessAccount {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub balance: Option<U256>,
    pub nonce: u64,
    #[serde(default, skip_serializing_if = "bytes_is_empty")]
    pub code: Bytes,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub storage: BTreeMap<B256, B256>,
}

#[derive(Debug, thiserror::Error)]
pub enum ProtocolError {
    #[error("input exceeds the {0} byte protocol limit")]
    TooLarge(usize),
    #[error("invalid JSON: {0}")]
    Json(#[from] serde_json::Error),
    #[error("unsupported witness schema {0:?}")]
    WitnessSchema(String),
    #[error("witness chainId must be non-zero")]
    ChainId,
    #[error("witness transaction is empty")]
    Transaction,
    #[error("witness prestate is empty")]
    Prestate,
}

impl ReplayWitness {
    pub fn decode_strict(bytes: &[u8]) -> Result<Self, ProtocolError> {
        if bytes.len() > MAX_WITNESS_BYTES {
            return Err(ProtocolError::TooLarge(MAX_WITNESS_BYTES));
        }
        let mut deserializer = serde_json::Deserializer::from_slice(bytes);
        let witness = Self::deserialize(&mut deserializer)?;
        deserializer.end()?;
        witness.validate()?;
        Ok(witness)
    }

    pub fn validate(&self) -> Result<(), ProtocolError> {
        if self.schema != WITNESS_SCHEMA {
            return Err(ProtocolError::WitnessSchema(self.schema.clone()));
        }
        if self.chain_id == 0 {
            return Err(ProtocolError::ChainId);
        }
        if self.transaction.is_empty() {
            return Err(ProtocolError::Transaction);
        }
        if self.prestate.is_empty() {
            return Err(ProtocolError::Prestate);
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn protocol_versions_are_frozen() {
        assert_eq!(TRACE_SCHEMA, "echoevm.trace.v1");
        assert_eq!(EVIDENCE_SCHEMA, "echoevm.evidence.v1");
        assert_eq!(WITNESS_SCHEMA, "echoevm.replay-witness.v1");
        assert_eq!(PROTOCOL_VERSION, 1);
    }

    #[test]
    fn witness_rejects_unknown_fields() {
        let input = br#"{"schema":"echoevm.replay-witness.v1","chainId":1,"blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","transactionIndex":0,"header":{},"transaction":"0x01","prestate":{"0x0000000000000000000000000000000000000000":{"nonce":0}},"extra":true}"#;
        assert!(ReplayWitness::decode_strict(input).is_err());
    }
}
