mod debug;

pub use debug::import_debug;

use alloy_primitives::{Address, B256, Bytes, U256, keccak256};
use alloy_trie::{EMPTY_ROOT_HASH, Nibbles, TrieAccount, proof::verify_proof};
use anyhow::{Context, Result, bail};
use echoevm_core::{
    ExecuteError, TransactionWitnessMaterialization, materialize_transaction_witness,
    replay_witness,
};
use echoevm_protocol::WitnessAccount;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::Path,
};

const PROOF_BUNDLE_SCHEMA: &str = "echoevm.witness-proofs.v1";

pub fn import_proof(
    input: &str,
    rpc_url: &str,
    output: Option<&Path>,
    proofs_output: Option<&Path>,
    blockhash_depth: u16,
) -> Result<()> {
    require_rpc_url(rpc_url)?;
    let hash = transaction_hash(input)?;
    let chain_id = quantity_u64(
        rpc(rpc_url, "eth_chainId", json!([]))?
            .as_str()
            .context("RPC chain id is not a quantity")?,
    )?;
    let tx = rpc(rpc_url, "eth_getTransactionByHash", json!([hash]))?;
    if tx.is_null() {
        bail!("transaction {hash} was not found");
    }
    let block_hash: B256 = tx
        .get("blockHash")
        .and_then(Value::as_str)
        .context("transaction is pending")?
        .parse()?;
    let transaction_index = quantity_u64(
        tx.get("transactionIndex")
            .and_then(Value::as_str)
            .context("missing transaction index")?,
    )?;
    let header = rpc(rpc_url, "eth_getBlockByHash", json!([block_hash, false]))?;
    let parent_hash: B256 = header
        .get("parentHash")
        .and_then(Value::as_str)
        .context("block header is missing parentHash")?
        .parse()?;
    let parent = rpc(rpc_url, "eth_getBlockByHash", json!([parent_hash, false]))?;
    if parent.is_null() {
        bail!("parent block {parent_hash} was not found");
    }
    let parent_number = parent
        .get("number")
        .and_then(Value::as_str)
        .context("parent block is missing number")?
        .to_owned();
    let parent_state_root: B256 = parent
        .get("stateRoot")
        .and_then(Value::as_str)
        .context("parent block is missing stateRoot")?
        .parse()?;

    let transaction_hashes = header
        .get("transactions")
        .and_then(Value::as_array)
        .context("block is missing transactions")?;
    let prefix_hashes = transaction_hashes
        .get(..=transaction_index as usize)
        .context("transaction index exceeds block transaction inventory")?;
    let mut prefix_transactions = Vec::with_capacity(prefix_hashes.len());
    let mut requested = BTreeMap::<Address, BTreeSet<B256>>::new();
    for value in prefix_hashes {
        let prefix_hash = value.as_str().context("block transaction is not a hash")?;
        let prefix_tx = rpc(rpc_url, "eth_getTransactionByHash", json!([prefix_hash]))?;
        seed_transaction_reads(rpc_url, &prefix_tx, &parent_number, &mut requested)?;
        let raw: Bytes = rpc(rpc_url, "eth_getRawTransactionByHash", json!([prefix_hash]))?
            .as_str()
            .context("RPC does not expose eth_getRawTransactionByHash")?
            .parse()?;
        prefix_transactions.push(raw);
    }
    if let Some(address) = header
        .get("miner")
        .or_else(|| header.get("beneficiary"))
        .and_then(Value::as_str)
    {
        requested.entry(address.parse()?).or_default();
    }

    let block_hashes = collect_block_hashes(rpc_url, parent, blockhash_depth)?;
    let mut discovery_rounds = 0_u8;
    let (witness, proofs) = loop {
        discovery_rounds += 1;
        if discovery_rounds > 64 {
            bail!("iterative witness discovery exceeded 64 rounds");
        }
        let mut prestate = BTreeMap::new();
        let mut proofs = Vec::new();
        for (address, storage_keys) in &requested {
            let proof: RpcProof = serde_json::from_value(rpc(
                rpc_url,
                "eth_getProof",
                json!([address, storage_keys, &parent_number]),
            )?)?;
            if proof.address != *address {
                bail!(
                    "eth_getProof returned address {} for requested {address}",
                    proof.address
                );
            }
            let code: Bytes = rpc(rpc_url, "eth_getCode", json!([address, &parent_number]))?
                .as_str()
                .context("eth_getCode returned a non-string")?
                .parse()?;
            let account = verify_rpc_proof(parent_state_root, &proof, &code)
                .with_context(|| format!("verify EIP-1186 proof for {address}"))?;
            prestate.insert(*address, account);
            proofs.push(proof);
        }
        match materialize_transaction_witness(TransactionWitnessMaterialization {
            chain_id,
            block_hash,
            header: header.clone(),
            transactions: &prefix_transactions,
            target_index: transaction_index as usize,
            parent_prestate: &prestate,
            block_hashes: block_hashes.clone(),
            source: Some(format!(
                "proof-verified parent-state acquisition with {} preceding transaction(s) replayed by EchoEVM at parent state root {parent_state_root}",
                transaction_index
            )),
        }) {
            Ok(witness) => {
                replay_witness(&witness, false)?;
                break (witness, proofs);
            }
            Err(ExecuteError::IncompleteWitness { accounts, storage }) => {
                let before = requested.values().map(BTreeSet::len).sum::<usize>() + requested.len();
                for address in accounts {
                    requested.entry(address).or_default();
                }
                for (address, slot) in storage {
                    requested.entry(address).or_default().insert(slot);
                }
                let after = requested.values().map(BTreeSet::len).sum::<usize>() + requested.len();
                if after == before {
                    bail!("iterative witness discovery made no progress");
                }
            }
            Err(error) => return Err(error.into()),
        }
    };
    write_json(output, &witness)?;

    if let Some(path) = proofs_output {
        let bundle = ProofBundle {
            schema: PROOF_BUNDLE_SCHEMA.into(),
            transaction_hash: hash.into(),
            block_hash,
            parent_block_hash: parent_hash,
            parent_state_root,
            accounts: proofs,
        };
        fs::write(path, serde_json::to_vec_pretty(&bundle)?)
            .with_context(|| format!("write {}", path.display()))?;
    }
    Ok(())
}

fn seed_transaction_reads(
    rpc_url: &str,
    transaction: &Value,
    parent_number: &str,
    requested: &mut BTreeMap<Address, BTreeSet<B256>>,
) -> Result<()> {
    if let Ok(value) = rpc(
        rpc_url,
        "eth_createAccessList",
        json!([access_list_call(transaction)?, parent_number]),
    ) && let Ok(access_result) = serde_json::from_value::<RpcAccessListResult>(value)
        && access_result.error.is_none()
    {
        for item in access_result.access_list {
            requested
                .entry(item.address)
                .or_default()
                .extend(item.storage_keys);
        }
    }
    for field in ["from", "to"] {
        if let Some(address) = transaction.get(field).and_then(Value::as_str) {
            requested.entry(address.parse()?).or_default();
        }
    }
    Ok(())
}

fn require_rpc_url(rpc_url: &str) -> Result<()> {
    if rpc_url.trim().is_empty() {
        bail!("--rpc-url or ETHEREUM_RPC_URL is required for witness acquisition");
    }
    Ok(())
}

fn write_json<T: Serialize>(output: Option<&Path>, value: &T) -> Result<()> {
    let bytes = serde_json::to_vec_pretty(value)?;
    if let Some(path) = output {
        fs::write(path, bytes).with_context(|| format!("write {}", path.display()))?;
    } else {
        println!("{}", String::from_utf8(bytes)?);
    }
    Ok(())
}

fn access_list_call(tx: &Value) -> Result<Value> {
    let source = tx
        .as_object()
        .context("eth_getTransactionByHash returned a non-object")?;
    let mut call = Map::new();
    for field in [
        "from",
        "to",
        "gas",
        "value",
        "input",
        "nonce",
        "type",
        "accessList",
        "chainId",
        "maxFeePerBlobGas",
        "blobVersionedHashes",
        "authorizationList",
    ] {
        if let Some(value) = source.get(field) {
            call.insert(field.into(), value.clone());
        }
    }
    if source.get("maxFeePerGas").is_some() {
        for field in ["maxFeePerGas", "maxPriorityFeePerGas"] {
            if let Some(value) = source.get(field) {
                call.insert(field.into(), value.clone());
            }
        }
    } else if let Some(value) = source.get("gasPrice") {
        call.insert("gasPrice".into(), value.clone());
    }
    Ok(Value::Object(call))
}

fn collect_block_hashes(
    rpc_url: &str,
    mut block: Value,
    depth: u16,
) -> Result<BTreeMap<String, B256>> {
    let mut hashes = BTreeMap::new();
    for _ in 0..depth {
        let number = quantity_u64(
            block
                .get("number")
                .and_then(Value::as_str)
                .context("historical block is missing number")?,
        )?;
        let hash: B256 = block
            .get("hash")
            .and_then(Value::as_str)
            .context("historical block is missing hash")?
            .parse()?;
        hashes.insert(number.to_string(), hash);
        if number == 0 {
            break;
        }
        let parent: B256 = block
            .get("parentHash")
            .and_then(Value::as_str)
            .context("historical block is missing parentHash")?
            .parse()?;
        block = rpc(rpc_url, "eth_getBlockByHash", json!([parent, false]))?;
        if block.is_null() {
            bail!("historical block {parent} required for BLOCKHASH was not found");
        }
    }
    Ok(hashes)
}

fn verify_rpc_proof(state_root: B256, proof: &RpcProof, code: &Bytes) -> Result<WitnessAccount> {
    if keccak256(code) != proof.code_hash {
        bail!(
            "code hash mismatch: proof commits to {}, fetched code hashes to {}",
            proof.code_hash,
            keccak256(code)
        );
    }
    let balance = quantity_u256(&proof.balance)?;
    let nonce = quantity_u64(&proof.nonce)?;
    let trie_account = TrieAccount {
        nonce,
        balance,
        storage_root: proof.storage_hash,
        code_hash: proof.code_hash,
    };
    let empty = nonce == 0
        && balance == U256::ZERO
        && proof.storage_hash == EMPTY_ROOT_HASH
        && proof.code_hash == alloy_trie::KECCAK_EMPTY;
    let expected = (!empty).then(|| alloy_rlp::encode(trie_account));
    let account_key = Nibbles::unpack(keccak256(proof.address));
    verify_proof(state_root, account_key, expected, &proof.account_proof)
        .map_err(|error| anyhow::anyhow!(error.to_string()))?;

    let mut storage = BTreeMap::new();
    for item in &proof.storage_proof {
        let slot_value = quantity_u256(&item.key)?;
        let slot = B256::from(slot_value.to_be_bytes::<32>());
        let value = quantity_u256(&item.value)?;
        let expected = (value != U256::ZERO).then(|| alloy_rlp::encode(value));
        verify_proof(
            proof.storage_hash,
            Nibbles::unpack(keccak256(slot)),
            expected,
            &item.proof,
        )
        .map_err(|error| anyhow::anyhow!(error.to_string()))?;
        if !empty {
            storage.insert(slot, B256::from(value.to_be_bytes::<32>()));
        }
    }
    Ok(WitnessAccount {
        exists: Some(!empty),
        balance: (!empty).then_some(balance),
        nonce,
        code: code.clone(),
        storage,
        storage_complete: empty,
    })
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct RpcAccessListResult {
    #[serde(default)]
    access_list: Vec<RpcAccessItem>,
    #[serde(default)]
    error: Option<String>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct RpcAccessItem {
    address: Address,
    #[serde(default)]
    storage_keys: Vec<B256>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct RpcProof {
    address: Address,
    account_proof: Vec<Bytes>,
    balance: String,
    code_hash: B256,
    nonce: String,
    storage_hash: B256,
    #[serde(default)]
    storage_proof: Vec<RpcStorageProof>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct RpcStorageProof {
    key: String,
    value: String,
    proof: Vec<Bytes>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct ProofBundle {
    schema: String,
    transaction_hash: String,
    block_hash: B256,
    parent_block_hash: B256,
    parent_state_root: B256,
    accounts: Vec<RpcProof>,
}

#[derive(serde::Deserialize)]
struct RpcAccount {
    balance: Option<String>,
    nonce: Option<String>,
    code: Option<String>,
    #[serde(default)]
    storage: BTreeMap<String, String>,
}

fn rpc(url: &str, method: &str, params: Value) -> Result<Value> {
    let response: Value = ureq::post(url)
        .send_json(json!({"jsonrpc":"2.0","id":1,"method":method,"params":params}))?
        .into_json()?;
    if let Some(error) = response.get("error") {
        bail!("RPC {method} failed: {error}");
    }
    response
        .get("result")
        .cloned()
        .context("RPC response is missing result")
}

fn transaction_hash(input: &str) -> Result<&str> {
    let value = input.trim();
    let hash = if value.starts_with("http://") || value.starts_with("https://") {
        value.rsplit('/').next().unwrap_or("")
    } else {
        value
    };
    if hash.len() != 66
        || !hash.starts_with("0x")
        || !hash[2..].bytes().all(|b| b.is_ascii_hexdigit())
    {
        bail!("enter a 32-byte transaction hash or Etherscan /tx/ URL");
    }
    Ok(hash)
}

fn quantity_u64(value: &str) -> Result<u64> {
    Ok(u64::from_str_radix(value.trim_start_matches("0x"), 16)?)
}
fn quantity_u256(value: &str) -> Result<U256> {
    Ok(U256::from_str_radix(value.trim_start_matches("0x"), 16)?)
}

#[cfg(test)]
mod tests;
