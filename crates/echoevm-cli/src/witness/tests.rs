use super::*;
use alloy_trie::{HashBuilder, proof::ProofRetainer};

#[test]
fn eip1559_access_call_does_not_mix_gas_price_models() {
    let call = access_list_call(&json!({
        "from": "0x0000000000000000000000000000000000000001",
        "gasPrice": "0x9",
        "maxFeePerGas": "0x8",
        "maxPriorityFeePerGas": "0x2",
        "hash": "ignored"
    }))
    .unwrap();
    assert!(call.get("gasPrice").is_none());
    assert_eq!(call["maxFeePerGas"], "0x8");
    assert!(call.get("hash").is_none());
}

#[test]
fn verifies_an_inclusion_account_proof() {
    let address = Address::with_last_byte(0x42);
    let key = Nibbles::unpack(keccak256(address));
    let account = TrieAccount {
        nonce: 1,
        balance: U256::from(7),
        storage_root: EMPTY_ROOT_HASH,
        code_hash: alloy_trie::KECCAK_EMPTY,
    };
    let encoded = alloy_rlp::encode(account);
    let mut builder = HashBuilder::default().with_proof_retainer(ProofRetainer::from_iter([key]));
    builder.add_leaf(key, &encoded);
    let root = builder.root();
    let nodes = builder
        .take_proof_nodes()
        .into_nodes_sorted()
        .into_iter()
        .map(|(_, node)| node)
        .collect();
    let proof = RpcProof {
        address,
        account_proof: nodes,
        balance: "0x7".into(),
        code_hash: alloy_trie::KECCAK_EMPTY,
        nonce: "0x1".into(),
        storage_hash: EMPTY_ROOT_HASH,
        storage_proof: Vec::new(),
    };
    let witness = verify_rpc_proof(root, &proof, &Bytes::new()).unwrap();
    assert_eq!(witness.nonce, 1);
    assert_eq!(witness.balance, Some(U256::from(7)));
}

#[test]
fn preserves_a_proved_absent_account() {
    let proof = RpcProof {
        address: Address::with_last_byte(0x43),
        account_proof: Vec::new(),
        balance: "0x0".into(),
        code_hash: alloy_trie::KECCAK_EMPTY,
        nonce: "0x0".into(),
        storage_hash: EMPTY_ROOT_HASH,
        storage_proof: Vec::new(),
    };
    let witness = verify_rpc_proof(EMPTY_ROOT_HASH, &proof, &Bytes::new()).unwrap();
    assert_eq!(witness.exists, Some(false));
    assert_eq!(witness.balance, None);
}
