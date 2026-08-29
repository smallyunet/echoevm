use super::{ExecutionResult, MAX_WITNESS_BYTES, ProtocolError, TestFork, WitnessAccount};
use crate::replay_witness::validate_witness_accounts;
use alloy_primitives::{Address, B256, Bytes};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

pub const BLOCK_WITNESS_SCHEMA: &str = "echoevm.block-witness.v1";
pub const BLOCK_RESULT_SCHEMA: &str = "echoevm.block-result.v1";

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct BlockWitness {
    pub schema: String,
    pub chain_id: u64,
    pub fork: TestFork,
    pub block_hash: B256,
    pub header: serde_json::Value,
    #[serde(default)]
    pub transactions: Vec<Bytes>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub withdrawals: Vec<BlockWithdrawal>,
    pub prestate: BTreeMap<Address, WitnessAccount>,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub block_hashes: BTreeMap<String, B256>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source: Option<String>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct BlockWithdrawal {
    pub index: u64,
    pub validator_index: u64,
    pub address: Address,
    pub amount: u64,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct BlockExecutionResult {
    pub schema: String,
    pub engine: String,
    pub engine_version: String,
    pub block_hash: String,
    pub block_number: u64,
    pub fork: String,
    pub transaction_count: usize,
    pub gas_used: u64,
    pub state_root: String,
    pub receipts_root: String,
    pub logs_bloom: String,
    #[serde(default)]
    pub transactions: Vec<ExecutionResult>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub warnings: Vec<String>,
}

impl BlockWitness {
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
        if self.schema != BLOCK_WITNESS_SCHEMA {
            return Err(ProtocolError::BlockWitnessSchema(self.schema.clone()));
        }
        if self.chain_id == 0 {
            return Err(ProtocolError::ChainId);
        }
        if self.prestate.is_empty() {
            return Err(ProtocolError::Prestate);
        }
        if self
            .transactions
            .iter()
            .any(|transaction| transaction.is_empty())
        {
            return Err(ProtocolError::Transaction);
        }
        validate_witness_accounts(&self.prestate)?;
        Ok(())
    }
}
