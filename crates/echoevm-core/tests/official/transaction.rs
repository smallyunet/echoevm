use super::*;

pub(super) fn build_transaction(
    unit: &Unit,
    case: &JsonPost,
    gate: Gate,
) -> Result<Transaction, String> {
    let data = decode_bytes(
        unit.transaction
            .data
            .get(case.indexes.data)
            .ok_or("data index")?,
    )?;
    let gas_limit = quantity_u64(
        unit.transaction
            .gas_limit
            .get(case.indexes.gas)
            .ok_or("gas index")?,
    )?;
    let value = quantity(
        unit.transaction
            .value
            .get(case.indexes.value)
            .ok_or("value index")?,
    )?;
    let base_fee = unit
        .env
        .current_base_fee
        .as_deref()
        .map(quantity)
        .transpose()?
        .unwrap_or_default();
    let gas_price = if let Some(price) = &unit.transaction.gas_price {
        quantity(price)?
    } else {
        let max_fee = quantity(
            unit.transaction
                .max_fee_per_gas
                .as_deref()
                .ok_or("missing maxFeePerGas")?,
        )?;
        let priority = unit
            .transaction
            .max_priority_fee_per_gas
            .as_deref()
            .map(quantity)
            .transpose()?
            .unwrap_or_default();
        max_fee.min(base_fee.saturating_add(priority))
    };
    let max_fee_per_gas = unit
        .transaction
        .max_fee_per_gas
        .as_deref()
        .or(unit.transaction.gas_price.as_deref())
        .map(quantity)
        .transpose()?
        .unwrap_or_default();
    let block_number = quantity_u64(&unit.env.current_number)?;
    let block_hashes = if block_number == 0 {
        BTreeMap::new()
    } else {
        BTreeMap::from([(block_number - 1, keccak256(b"0"))])
    };
    Ok(Transaction {
        caller: parse_address(&unit.transaction.sender)?,
        to: (!unit.transaction.to.is_empty() && unit.transaction.to != "0x")
            .then(|| parse_address(&unit.transaction.to))
            .transpose()?,
        value,
        data,
        gas_limit,
        gas_price,
        max_fee_per_gas,
        max_priority_fee_per_gas: unit
            .transaction
            .max_priority_fee_per_gas
            .as_deref()
            .map(quantity)
            .transpose()?,
        nonce: quantity_u64(&unit.transaction.nonce)?,
        access_list: unit
            .transaction
            .access_lists
            .get(case.indexes.data)
            .into_iter()
            .flatten()
            .map(|item| {
                Ok((
                    parse_address(&item.address)?,
                    item.storage_keys
                        .iter()
                        .map(|key| quantity(key))
                        .collect::<Result<_, _>>()?,
                ))
            })
            .collect::<Result<_, String>>()?,
        authorization_list: unit
            .transaction
            .authorization_list
            .as_deref()
            .unwrap_or_default()
            .iter()
            .map(|authorization| {
                Ok(Authorization {
                    chain_id: quantity(&authorization.chain_id)?,
                    delegate: parse_address(&authorization.address)?,
                    nonce: quantity_u64(&authorization.nonce)?,
                    y_parity: quantity_u64(
                        authorization
                            .y_parity
                            .as_deref()
                            .or(authorization.v.as_deref())
                            .ok_or("authorization y parity")?,
                    )? as u8,
                    r: quantity(&authorization.r)?,
                    s: quantity(&authorization.s)?,
                })
            })
            .collect::<Result<_, String>>()?,
        set_code: unit.transaction.authorization_list.is_some(),
        max_fee_per_blob_gas: unit
            .transaction
            .max_fee_per_blob_gas
            .as_deref()
            .map(quantity)
            .transpose()?,
        fork: match gate.fork {
            "Cancun" => Fork::Cancun,
            "Prague" => Fork::Prague,
            "Osaka" => Fork::Osaka,
            _ => unreachable!(),
        },
        environment: BlockEnv {
            chain_id: 1,
            block_number,
            timestamp: quantity_u64(&unit.env.current_timestamp)?,
            coinbase: parse_address(&unit.env.current_coinbase)?,
            block_gas_limit: quantity_u64(&unit.env.current_gas_limit)?,
            base_fee,
            prevrandao: unit
                .env
                .current_random
                .as_deref()
                .or(unit.env.current_difficulty.as_deref())
                .map(quantity)
                .transpose()?
                .unwrap_or_default(),
            blob_base_fee: fake_exponential(
                1,
                unit.env
                    .current_excess_blob_gas
                    .as_deref()
                    .map(quantity_u64)
                    .transpose()?
                    .unwrap_or_default(),
                3_338_477,
            ),
            block_hashes,
            blob_hashes: unit
                .transaction
                .blob_versioned_hashes
                .iter()
                .map(|value| value.parse::<B256>().map_err(|error| error.to_string()))
                .collect::<Result<_, _>>()?,
        },
        trace: false,
    })
}

pub(super) fn decode_state(input: &BTreeMap<String, JsonAccount>) -> WorldState {
    let mut world = WorldState::default();
    for (address, account) in input {
        world.accounts.insert(
            parse_address(address).expect("fixture address"),
            decode_account(account),
        );
    }
    world
}

pub(super) fn decode_account(input: &JsonAccount) -> Account {
    Account {
        nonce: quantity_u64(&input.nonce).expect("fixture nonce"),
        balance: quantity(&input.balance).expect("fixture balance"),
        code: decode_bytes(&input.code).expect("fixture code"),
        storage: input
            .storage
            .iter()
            .filter_map(|(slot, value)| {
                let value = quantity(value).expect("value");
                (!value.is_zero()).then(|| (quantity(slot).expect("slot"), value))
            })
            .collect(),
    }
}

pub(super) fn parse_address(input: &str) -> Result<Address, String> {
    input.parse().map_err(|error| format!("{error}"))
}
pub(super) fn decode_bytes(input: &str) -> Result<Vec<u8>, String> {
    hex::decode(input.trim_start_matches("0x")).map_err(|error| error.to_string())
}
pub(super) fn quantity(input: &str) -> Result<U256, String> {
    U256::from_str_radix(input.trim_start_matches("0x"), 16).map_err(|error| error.to_string())
}
pub(super) fn quantity_u64(input: &str) -> Result<u64, String> {
    quantity(input)?
        .try_into()
        .map_err(|_| format!("quantity exceeds u64: {input}"))
}

pub(super) fn fake_exponential(factor: u64, numerator: u64, denominator: u64) -> U256 {
    let mut output = U256::ZERO;
    let mut accumulator = U256::from(factor) * U256::from(denominator);
    let mut index = 1u64;
    while !accumulator.is_zero() {
        output += accumulator;
        accumulator = accumulator * U256::from(numerator) / U256::from(denominator * index);
        index += 1;
    }
    output / U256::from(denominator)
}
