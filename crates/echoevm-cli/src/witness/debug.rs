use super::*;
use echoevm_protocol::{ReplayWitness, WITNESS_SCHEMA};

pub fn import_debug(input: &str, rpc_url: &str, output: Option<&Path>) -> Result<()> {
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
                    exists: Some(true),
                    balance: account.balance.as_deref().map(quantity_u256).transpose()?,
                    nonce: account
                        .nonce
                        .as_deref()
                        .map(quantity_u64)
                        .transpose()?
                        .unwrap_or_default(),
                    code: account.code.unwrap_or_else(|| "0x".into()).parse()?,
                    storage,
                    storage_complete: false,
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
    write_json(output, &witness)
}
