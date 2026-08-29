use super::system::{protocol_address, system_address};
use super::*;
use alloy_consensus::{SignableTransaction, TxEip1559};
use alloy_eips::{Encodable2718, eip2930::AccessList};
use alloy_primitives::{Bytes, Signature, TxKind};
use k256::ecdsa::SigningKey;

fn signed_transfer(nonce: u64, recipient: Address) -> (alloy_primitives::Bytes, Address) {
    let key = SigningKey::from_bytes((&[7u8; 32]).into()).unwrap();
    let transaction = TxEip1559 {
        chain_id: 1,
        nonce,
        gas_limit: 21_000,
        max_fee_per_gas: 2,
        max_priority_fee_per_gas: 1,
        to: TxKind::Call(recipient),
        value: U256::from(1),
        access_list: AccessList::default(),
        input: Bytes::new(),
    };
    let (signature, recovery_id) = key
        .sign_prehash_recoverable(transaction.signature_hash().as_ref())
        .unwrap();
    let signature: Signature = (signature, recovery_id).into();
    let envelope: TxEnvelope = transaction.into_signed(signature).into();
    let sender = envelope.recover_signer().unwrap();
    (envelope.encoded_2718().into(), sender)
}

#[test]
fn materializes_intermediate_prestate_for_later_transaction() {
    let recipient = Address::from([0x22; 20]);
    let (first, sender) = signed_transfer(0, recipient);
    let (second, _) = signed_transfer(1, recipient);
    let header = Header {
        number: 20_000_000,
        gas_limit: 1_000_000,
        timestamp: crate::MAINNET_CANCUN_TIME,
        base_fee_per_gas: Some(1),
        beneficiary: Address::from([0x33; 20]),
        ..Default::default()
    };
    let block_hash = header.hash_slow();
    let prestate = BTreeMap::from([
        (
            sender,
            WitnessAccount {
                exists: Some(true),
                balance: Some(U256::from(1_000_000)),
                ..Default::default()
            },
        ),
        (recipient, WitnessAccount::default()),
        (header.beneficiary, WitnessAccount::default()),
    ]);
    let transactions = [first, second];
    let witness = materialize_transaction_witness(TransactionWitnessMaterialization {
        chain_id: 1,
        block_hash,
        header: serde_json::to_value(&header).unwrap(),
        transactions: &transactions,
        target_index: 1,
        parent_prestate: &prestate,
        block_hashes: BTreeMap::new(),
        source: Some("unit-test".into()),
    })
    .unwrap();

    assert_eq!(witness.transaction_index, 1);
    assert_eq!(witness.prestate[&sender].nonce, 1);
    assert_eq!(witness.prestate[&recipient].balance, Some(U256::from(1)));
    let replay = crate::replay_witness(&witness, false).unwrap();
    assert_eq!(replay.execution.status, ExecutionStatus::Success);
}

#[test]
fn prague_history_system_call_stores_parent_hash() {
    let target = protocol_address("0x0000f90827f1c53a10cb7a02335b175320002935");
    let mut world = WorldState::default();
    world.account_mut(target).code = hex::decode(
        "3373fffffffffffffffffffffffffffffffffffffffe14604657602036036042575f35600143038111604257611fff81430311604257611fff9006545f5260205ff35b5f5ffd5b5f35611fff60014303065500",
    )
    .unwrap();
    let parent_hash = B256::from([0x42; 32]);
    let environment = Environment {
        block_number: 1,
        timestamp: crate::MAINNET_PRAGUE_TIME,
        ..Default::default()
    };
    engine::system_call(
        &mut world,
        system_address(),
        target,
        parent_hash.to_vec(),
        30_000_000,
        Fork::Prague,
        environment,
    )
    .unwrap();
    assert_eq!(
        world.storage(target, U256::ZERO),
        U256::from_be_bytes(parent_hash.0)
    );
}
