use alloy_consensus::Header;
use echoevm_core::{MAINNET_CANCUN_TIME, decode_hex};
use echoevm_protocol::{ReplayWitness, WITNESS_SCHEMA, WitnessAccount};
use revm::primitives::{Address, B256, U256};
use std::{collections::BTreeMap, env, fs, path::PathBuf};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let output = env::args_os()
        .nth(1)
        .map(PathBuf::from)
        .ok_or("output path is required")?;
    let raw = decode_hex(
        "02f86f0102843b9aca0085029e7822d68298f094d9e1459a7a482635700cbc20bbaf52d495ab9c9680841b55ba3ac080a0c199674fcb29f353693dd779c017823b954b3c69dffa3cd6b2a6ff7888798039a028ca912de909e7e6cdef9cdcaf24c54dd8c1032946dfa1d85c206b32a9064fe8",
    )?;
    let sender: Address = "0x001e2b7de757ba469a57bf6b23d982458a07efce".parse()?;
    let recipient: Address = "0xd9e1459a7a482635700cbc20bbaf52d495ab9c96".parse()?;
    let header = Header {
        number: 19_500_000,
        gas_limit: 30_000_000,
        timestamp: MAINNET_CANCUN_TIME,
        base_fee_per_gas: Some(1),
        withdrawals_root: Some(B256::ZERO),
        blob_gas_used: Some(0),
        excess_blob_gas: Some(0),
        parent_beacon_block_root: Some(B256::ZERO),
        ..Default::default()
    };
    let witness = ReplayWitness {
        schema: WITNESS_SCHEMA.into(),
        chain_id: 1,
        block_hash: header.hash_slow(),
        transaction_index: 0,
        header: serde_json::to_value(header)?,
        transaction: raw.into(),
        prestate: BTreeMap::from([
            (
                sender,
                WitnessAccount {
                    balance: Some(U256::from(10_000_000_000_000_000_000_u128)),
                    nonce: 2,
                    ..Default::default()
                },
            ),
            (recipient, WitnessAccount::default()),
        ]),
        block_hashes: BTreeMap::new(),
        source: Some("echoevm-rust-wasm-smoke".into()),
    };
    fs::write(output, serde_json::to_vec_pretty(&witness)?)?;
    Ok(())
}
