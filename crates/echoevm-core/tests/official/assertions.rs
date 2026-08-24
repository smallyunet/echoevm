use super::*;

pub(super) fn assert_commitments(
    path: &Path,
    name: &str,
    index: usize,
    world: &WorldState,
    result: &echoevm_protocol::ExecutionResult,
    case: &JsonPost,
) {
    let expected_root = case.hash.parse::<B256>().expect("fixture state root");
    assert_eq!(
        world.state_root(),
        expected_root,
        "{}::{name}[{index}] state root",
        path.display()
    );
    let expected_logs = case.logs.parse::<B256>().expect("fixture logs hash");
    assert_eq!(
        keccak256(alloy_rlp::encode(&world.logs)),
        expected_logs,
        "{}::{name}[{index}] logs hash",
        path.display()
    );
    let receipt = case.receipt.as_ref().expect("accepted fixture receipt");
    assert_eq!(
        result.gas_used,
        quantity_u64(&receipt.cumulative_gas_used).expect("receipt gas"),
        "{}::{name}[{index}] receipt gas",
        path.display()
    );
    assert_eq!(
        result.status == ExecutionStatus::Success,
        receipt.status,
        "{}::{name}[{index}] receipt status",
        path.display()
    );
}

pub(super) fn validate_txbytes(
    case: &JsonPost,
    transaction: &JsonTransaction,
) -> Result<(), String> {
    let bytes = decode_bytes(&case.txbytes)?;
    let mut raw = bytes.as_slice();
    let envelope = TxEnvelope::decode_2718(&mut raw)
        .map_err(|error| format!("InvalidSignatureVrs: decode envelope: {error}"))?;
    if !raw.is_empty() {
        return Err("InvalidSignatureVrs: trailing transaction bytes".into());
    }
    if envelope.encoded_2718() != bytes {
        return Err("InvalidSignatureVrs: non-canonical transaction encoding".into());
    }
    if envelope.chain_id().is_some_and(|chain_id| chain_id != 1) {
        return Err(format!(
            "InvalidChainId: transaction chain id {:?} != fixture chain id 1",
            envelope.chain_id()
        ));
    }
    let signer = envelope
        .recover_signer()
        .map_err(|error| format!("InvalidSignatureVrs: recover sender: {error}"))?;
    let expected = parse_address(&transaction.sender)?;
    if signer != expected {
        return Err(format!(
            "InvalidSignatureVrs: recovered sender {signer} != fixture sender {expected}"
        ));
    }
    Ok(())
}

pub(super) fn assert_exception(
    path: &Path,
    name: &str,
    index: usize,
    expected: &str,
    actual: &str,
    unit: &Unit,
    case: &JsonPost,
) {
    let category = match actual.split(':').next().unwrap_or(actual) {
        "GasLimitExceedsBlockGasLimit" => "GAS_ALLOWANCE_EXCEEDED",
        "GasLimitExceedsMaximum" => "GAS_LIMIT_EXCEEDS_MAXIMUM",
        "CreateInitCodeSizeLimit" => "INITCODE_SIZE_EXCEEDED",
        "InsufficientFunds" => "INSUFFICIENT_ACCOUNT_FUNDS",
        "UpfrontCostOverflow" | "GasPaymentOverflow" | "BlobGasPaymentOverflow" => {
            "GASLIMIT_PRICE_PRODUCT_OVERFLOW"
        }
        "BlobGasPriceTooLow" => "INSUFFICIENT_MAX_FEE_PER_BLOB_GAS",
        "MaxFeePerGasBelowBaseFee" => "INSUFFICIENT_MAX_FEE_PER_GAS",
        "CalldataFloorGasTooLow" => "INTRINSIC_GAS_BELOW_FLOOR_GAS_COST",
        "IntrinsicGasTooLow" => "INTRINSIC_GAS_TOO_LOW",
        "InvalidChainId" => "INVALID_CHAINID",
        "InvalidSignatureVrs" => {
            if unit.transaction.max_fee_per_blob_gas.is_some() && unit.transaction.to.is_empty() {
                "TYPE_3_TX_CONTRACT_CREATION"
            } else if unit.transaction.authorization_list.is_some()
                && unit.transaction.to.is_empty()
            {
                "TYPE_4_TX_CONTRACT_CREATION"
            } else if let Some(gas_price) = unit.transaction.gas_price.as_deref() {
                let gas_price = quantity(gas_price).expect("fixture gas price");
                let gas_limit = quantity(&unit.transaction.gas_limit[case.indexes.gas])
                    .expect("fixture gas limit");
                if gas_price.checked_mul(gas_limit).is_none() {
                    "GASLIMIT_PRICE_PRODUCT_OVERFLOW"
                } else {
                    "INVALID_SIGNATURE_VRS"
                }
            } else {
                "INVALID_SIGNATURE_VRS"
            }
        }
        "NonceOverflow" => "NONCE_IS_MAX",
        "NonceMismatch" => {
            let sender = parse_address(&unit.transaction.sender).expect("fixture sender");
            let state_nonce = unit
                .pre
                .get(&sender.to_string().to_lowercase())
                .map(|account| quantity_u64(&account.nonce).expect("state nonce"))
                .unwrap_or_default();
            let tx_nonce = quantity_u64(&unit.transaction.nonce).expect("tx nonce");
            if tx_nonce > state_nonce {
                "NONCE_MISMATCH_TOO_HIGH"
            } else {
                "NONCE_MISMATCH_TOO_LOW"
            }
        }
        "PriorityFeeAboveMaxFee" => "PRIORITY_GREATER_THAN_MAX_FEE_PER_GAS",
        "SenderNotExternallyOwned" => "SENDER_NOT_EOA",
        "TooManyBlobs" => "TYPE_3_TX_BLOB_COUNT_EXCEEDED",
        "BlobGasAllowanceExceeded" => "TYPE_3_TX_MAX_BLOB_GAS_ALLOWANCE_EXCEEDED",
        "BlobTransactionContractCreation" => "TYPE_3_TX_CONTRACT_CREATION",
        "InvalidBlobVersionedHash" => "TYPE_3_TX_INVALID_BLOB_VERSIONED_HASH",
        "BlobTransactionMissingBlobHashes" => "TYPE_3_TX_ZERO_BLOBS",
        "EmptyAuthorizationList" => "TYPE_4_EMPTY_AUTHORIZATION_LIST",
        "SetCodeTransactionContractCreation" => "TYPE_4_TX_CONTRACT_CREATION",
        other => panic!(
            "{}::{name}[{index}] unmapped rejection {other}: {actual}",
            path.display()
        ),
    };
    let category_matches = expected
        .split('|')
        .any(|candidate| candidate.ends_with(category))
        || (actual == "CalldataFloorGasTooLow"
            && expected
                .split('|')
                .any(|candidate| candidate.ends_with("INTRINSIC_GAS_TOO_LOW")));
    assert!(
        category_matches,
        "{}::{name}[{index}] expected {expected}, got {actual} ({category})",
        path.display()
    );
}

pub(super) fn assert_rejected_commitments(
    path: &Path,
    name: &str,
    index: usize,
    world: &WorldState,
    case: &JsonPost,
) {
    assert_eq!(
        world.state_root(),
        case.hash.parse::<B256>().expect("rejected state root"),
        "{}::{name}[{index}] rejected state root",
        path.display()
    );
    assert_eq!(
        keccak256(alloy_rlp::encode(&world.logs)),
        case.logs.parse::<B256>().expect("rejected logs hash"),
        "{}::{name}[{index}] rejected logs hash",
        path.display()
    );
}

pub(super) fn assert_state(
    path: &Path,
    name: &str,
    index: usize,
    actual: &WorldState,
    expected: &BTreeMap<String, JsonAccount>,
) {
    let expected = decode_state(expected);
    let refund = actual.refund;
    let actual: BTreeMap<_, _> = actual
        .accounts
        .iter()
        .filter(|(_, account)| !account.is_empty())
        .map(|(address, account)| (*address, account.clone()))
        .collect();
    let expected: BTreeMap<_, _> = expected
        .accounts
        .into_iter()
        .filter(|(_, account)| !account.is_empty())
        .collect();
    if actual != expected {
        let addresses = actual
            .keys()
            .chain(expected.keys())
            .copied()
            .collect::<std::collections::BTreeSet<_>>();
        let mut mismatches = Vec::new();
        for address in addresses {
            let actual_account = actual.get(&address);
            let expected_account = expected.get(&address);
            if actual_account != expected_account {
                mismatches.push(format!(
                    "{address}: actual={actual_account:?} expected={expected_account:?}"
                ));
            }
        }
        panic!(
            "{}::{name}[{index}] state mismatches: {}; refund={refund}",
            path.display(),
            mismatches.join("; ")
        );
    }
}
