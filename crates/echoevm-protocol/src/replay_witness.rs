use super::*;

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
    pub exists: Option<bool>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub balance: Option<U256>,
    pub nonce: u64,
    #[serde(default, skip_serializing_if = "bytes_is_empty")]
    pub code: Bytes,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub storage: BTreeMap<B256, B256>,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub storage_complete: bool,
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
        validate_witness_accounts(&self.prestate)?;
        Ok(())
    }
}

pub(crate) fn validate_witness_accounts(
    accounts: &BTreeMap<Address, WitnessAccount>,
) -> Result<(), ProtocolError> {
    if accounts.values().any(|account| {
        account.exists == Some(false)
            && (account.balance.is_some()
                || account.nonce != 0
                || !account.code.is_empty()
                || !account.storage.is_empty())
    }) {
        return Err(ProtocolError::AbsentAccountState);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn absent_account_cannot_carry_state() {
        let account = WitnessAccount {
            exists: Some(false),
            balance: Some(U256::ZERO),
            ..Default::default()
        };
        let error =
            validate_witness_accounts(&BTreeMap::from([(Address::ZERO, account)])).unwrap_err();
        assert!(matches!(error, ProtocolError::AbsentAccountState));
    }
}
