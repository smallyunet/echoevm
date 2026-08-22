use alloy_primitives::{Address, B256, Bytes, U256};
use anyhow::{Context, Result, bail};
use echoevm_protocol::{ReplayWitness, WITNESS_SCHEMA, WitnessAccount};
use serde_json::{Value, json};
use std::{collections::BTreeMap, fs, path::Path};

pub fn import_debug(input: &str, rpc_url: &str, output: Option<&Path>) -> Result<()> {
    if rpc_url.trim().is_empty() {
        bail!(
            "--rpc-url or ETHEREUM_RPC_URL is required for the optional witness acquisition adapter"
        );
    }
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
    let raw: Bytes = rpc(rpc_url, "eth_getRawTransactionByHash", json!([hash]))?
        .as_str()
        .context("RPC does not expose the raw transaction")?
        .parse()?;
    let prestate = rpc(
        rpc_url,
        "debug_traceTransaction",
        json!([hash, {"tracer":"prestateTracer", "tracerConfig":{"diffMode":false}}]),
    )?;
    let accounts = prestate
        .as_object()
        .context("prestateTracer returned a non-object")?
        .iter()
        .map(|(address, value)| {
            let account: RpcAccount = serde_json::from_value(value.clone())?;
            let storage = account
                .storage
                .into_iter()
                .map(|(slot, value)| Ok((slot.parse()?, value.parse()?)))
                .collect::<Result<_>>()?;
            Ok((
                address.parse::<Address>()?,
                WitnessAccount {
                    balance: account.balance.as_deref().map(quantity_u256).transpose()?,
                    nonce: account
                        .nonce
                        .as_deref()
                        .map(quantity_u64)
                        .transpose()?
                        .unwrap_or_default(),
                    code: account.code.unwrap_or_else(|| "0x".into()).parse()?,
                    storage,
                },
            ))
        })
        .collect::<Result<BTreeMap<_, _>>>()?;
    let witness = ReplayWitness {
        schema: WITNESS_SCHEMA.into(),
        chain_id,
        block_hash,
        transaction_index,
        header,
        transaction: raw,
        prestate: accounts,
        block_hashes: BTreeMap::new(),
        source: Some("debug_traceTransaction/prestateTracer acquisition adapter".into()),
    };
    witness.validate()?;
    let bytes = serde_json::to_vec_pretty(&witness)?;
    if let Some(path) = output {
        fs::write(path, bytes).with_context(|| format!("write {}", path.display()))?;
    } else {
        println!("{}", String::from_utf8(bytes)?);
    }
    Ok(())
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
