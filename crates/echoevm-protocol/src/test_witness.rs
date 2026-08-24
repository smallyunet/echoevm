use super::*;

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub enum TestFork {
    Cancun,
    Prague,
    #[default]
    Osaka,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct TestExpectation {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub status: Option<ExecutionStatus>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub return_data: Option<Bytes>,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub storage: BTreeMap<B256, B256>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct TestSourceLocation {
    pub pc: u64,
    pub file: String,
    pub start: usize,
    pub length: usize,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct TestSourceMetadata {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub file: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub contract: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub function: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub test: Option<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub locations: Vec<TestSourceLocation>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct TestAccount {
    #[serde(default)]
    pub balance: U256,
    #[serde(default)]
    pub nonce: u64,
    #[serde(default, skip_serializing_if = "bytes_is_empty")]
    pub code: Bytes,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub storage: BTreeMap<B256, B256>,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct TestEnvironment {
    #[serde(default = "default_chain_id")]
    pub chain_id: u64,
    #[serde(default)]
    pub block_number: u64,
    #[serde(default)]
    pub timestamp: u64,
    #[serde(default)]
    pub coinbase: Address,
    #[serde(default = "default_block_gas_limit")]
    pub block_gas_limit: u64,
    #[serde(default)]
    pub base_fee: U256,
    #[serde(default)]
    pub prevrandao: U256,
    #[serde(default)]
    pub blob_base_fee: U256,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub block_hashes: BTreeMap<u64, B256>,
}

impl Default for TestEnvironment {
    fn default() -> Self {
        Self {
            chain_id: default_chain_id(),
            block_number: 0,
            timestamp: 0,
            coinbase: Address::ZERO,
            block_gas_limit: default_block_gas_limit(),
            base_fee: U256::ZERO,
            prevrandao: U256::ZERO,
            blob_base_fee: U256::ZERO,
            block_hashes: BTreeMap::new(),
        }
    }
}

fn default_chain_id() -> u64 {
    1
}
fn default_block_gas_limit() -> u64 {
    30_000_000
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct TestExecutionContext {
    pub caller: Address,
    pub target: Address,
    #[serde(default)]
    pub value: U256,
    #[serde(default)]
    pub gas_price: U256,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub accounts: BTreeMap<Address, TestAccount>,
    #[serde(default)]
    pub environment: TestEnvironment,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct TestWitness {
    pub schema: String,
    pub name: String,
    pub bytecode: Bytes,
    #[serde(default, skip_serializing_if = "bytes_is_empty")]
    pub calldata: Bytes,
    pub gas_limit: u64,
    #[serde(default)]
    pub fork: TestFork,
    #[serde(default)]
    pub expectation: TestExpectation,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub requires: Vec<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source: Option<TestSourceMetadata>,
    /// Explicit transaction/state context. When absent, the legacy isolated
    /// call executor is used for backwards compatibility.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub context: Option<TestExecutionContext>,
}

impl TestWitness {
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
        if self.schema != TEST_WITNESS_SCHEMA {
            return Err(ProtocolError::TestWitnessSchema(self.schema.clone()));
        }
        if self.bytecode.is_empty() {
            return Err(ProtocolError::TestBytecode);
        }
        if self.name.trim().is_empty() {
            return Err(ProtocolError::TestName);
        }
        if self.gas_limit == 0 {
            return Err(ProtocolError::TestGasLimit);
        }
        if !self.requires.is_empty() {
            return Err(ProtocolError::UnsupportedCapabilities(
                self.requires.join(", "),
            ));
        }
        if let Some(context) = &self.context {
            if context.caller == context.target {
                return Err(ProtocolError::TestContext(
                    "caller and target must be different accounts".into(),
                ));
            }
            if !context.accounts.contains_key(&context.caller) {
                return Err(ProtocolError::TestContext(format!(
                    "accounts is missing caller {}",
                    context.caller
                )));
            }
            if context.environment.block_gas_limit < self.gas_limit {
                return Err(ProtocolError::TestContext(
                    "environment.blockGasLimit is below gasLimit".into(),
                ));
            }
            if context.gas_price < context.environment.base_fee {
                return Err(ProtocolError::TestContext(
                    "gasPrice is below environment.baseFee".into(),
                ));
            }
            if context
                .accounts
                .get(&context.target)
                .is_some_and(|account| !account.code.is_empty() && account.code != self.bytecode)
            {
                return Err(ProtocolError::TestContext(
                    "accounts[target].code conflicts with bytecode".into(),
                ));
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fails_closed_for_required_capabilities() {
        let input = br#"{"schema":"echoevm.test-witness.v1","name":"cheatcode","bytecode":"0x00","gasLimit":100000,"requires":["foundry-cheatcodes"]}"#;
        let error = TestWitness::decode_strict(input).unwrap_err();
        assert!(error.to_string().contains("unsupported-capability"));
    }

    #[test]
    fn rejects_unknown_fields() {
        let input = br#"{"schema":"echoevm.test-witness.v1","name":"strict","bytecode":"0x00","gasLimit":100000,"unknown":true}"#;
        assert!(TestWitness::decode_strict(input).is_err());
    }

    #[test]
    fn accepts_explicit_state_context() {
        let input = br#"{
          "schema":"echoevm.test-witness.v1",
          "name":"stateful",
          "bytecode":"0x60005400",
          "gasLimit":100000,
          "context":{
            "caller":"0x1000000000000000000000000000000000000001",
            "target":"0x2000000000000000000000000000000000000002",
            "accounts":{
              "0x1000000000000000000000000000000000000001":{"balance":"0x100000"},
              "0x2000000000000000000000000000000000000002":{"storage":{"0x0000000000000000000000000000000000000000000000000000000000000000":"0x000000000000000000000000000000000000000000000000000000000000002a"}}
            }
          }
        }"#;
        let witness = TestWitness::decode_strict(input).unwrap();
        assert!(witness.context.is_some());
    }
}
