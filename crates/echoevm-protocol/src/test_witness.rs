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
}
