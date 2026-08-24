//! Transaction validation, state transition, deployment, and result normalization.

use super::*;

pub fn execute(request: Request) -> ExecutionResult {
    with_evm_stack(move || execute_inner(request))
}

pub(super) fn execute_inner(request: Request) -> ExecutionResult {
    let mut state = WorldState::default();
    state.begin_transaction();
    warm_precompiles(&mut state, request.fork);
    let address = Address::ZERO;
    let trace = request.trace;
    let gas_limit = request.gas_limit;
    let mut vm = Machine::new(request, &mut state, address);
    let halt = vm.run();
    let remaining = vm.gas;
    let steps = std::mem::take(&mut vm.steps);
    drop(vm);
    result_from_halt(halt, gas_limit, remaining, steps, trace, &state, address)
}

pub fn deploy(initcode: Vec<u8>, gas_limit: u64, fork: Fork, trace: bool) -> ExecutionResult {
    with_evm_stack(move || deploy_inner(initcode, gas_limit, fork, trace))
}

pub(super) fn deploy_inner(
    initcode: Vec<u8>,
    gas_limit: u64,
    fork: Fork,
    trace: bool,
) -> ExecutionResult {
    let mut state = WorldState::default();
    state.begin_transaction();
    warm_precompiles(&mut state, fork);
    let address = Address::from([0x10; 20]);
    let mut vm = Machine::new(
        Request {
            bytecode: initcode,
            calldata: Vec::new(),
            gas_limit,
            fork,
            trace,
        },
        &mut state,
        address,
    );
    let halt = vm.run();
    let remaining = vm.gas;
    let steps = std::mem::take(&mut vm.steps);
    drop(vm);
    if let Halt::Return(code) = &halt {
        state.account_mut(address).code = code.clone();
    }
    result_from_halt(halt, gas_limit, remaining, steps, trace, &state, address)
}

pub fn deploy_and_call(
    initcode: Vec<u8>,
    calldata: Vec<u8>,
    gas_limit: u64,
    fork: Fork,
    trace: bool,
) -> Result<ExecutionResult, &'static str> {
    with_evm_stack(move || deploy_and_call_inner(initcode, calldata, gas_limit, fork, trace))
}

pub(super) fn deploy_and_call_inner(
    initcode: Vec<u8>,
    calldata: Vec<u8>,
    gas_limit: u64,
    fork: Fork,
    trace: bool,
) -> Result<ExecutionResult, &'static str> {
    let mut state = WorldState::default();
    state.begin_transaction();
    warm_precompiles(&mut state, fork);
    let address = Address::from([0x10; 20]);
    let mut constructor = Machine::new(
        Request {
            bytecode: initcode,
            calldata: Vec::new(),
            gas_limit,
            fork,
            trace: false,
        },
        &mut state,
        address,
    );
    let deployed = constructor.run();
    drop(constructor);
    let runtime = match deployed {
        Halt::Return(code) => code,
        _ => return Err("contract deployment did not succeed"),
    };
    state.account_mut(address).code = runtime.clone();
    state.transient.clear();
    let mut call = Machine::new(
        Request {
            bytecode: runtime,
            calldata,
            gas_limit,
            fork,
            trace,
        },
        &mut state,
        address,
    );
    let halt = call.run();
    let remaining = call.gas;
    let steps = std::mem::take(&mut call.steps);
    drop(call);
    Ok(result_from_halt(
        halt, gas_limit, remaining, steps, trace, &state, address,
    ))
}

pub fn transact(
    state: &mut WorldState,
    transaction: Transaction,
) -> Result<ExecutionResult, &'static str> {
    with_evm_stack(move || transact_inner(state, transaction))
}

pub(super) fn transact_inner(
    state: &mut WorldState,
    transaction: Transaction,
) -> Result<ExecutionResult, &'static str> {
    if transaction.gas_limit > transaction.environment.block_gas_limit {
        return Err("GasLimitExceedsBlockGasLimit");
    }
    if transaction.fork == Fork::Osaka && transaction.gas_limit > OSAKA_TRANSACTION_GAS_LIMIT {
        return Err("GasLimitExceedsMaximum");
    }
    if transaction.to.is_none() && transaction.data.len() > 49_152 {
        return Err("CreateInitCodeSizeLimit");
    }
    if transaction.set_code {
        if transaction.fork == Fork::Cancun {
            return Err("SetCodeTransactionBeforePrague");
        }
        if transaction.to.is_none() {
            return Err("SetCodeTransactionContractCreation");
        }
        if transaction.authorization_list.is_empty() {
            return Err("EmptyAuthorizationList");
        }
        if transaction.max_fee_per_blob_gas.is_some()
            || !transaction.environment.blob_hashes.is_empty()
        {
            return Err("SetCodeTransactionWithBlobs");
        }
    }
    if transaction.max_fee_per_blob_gas.is_some() {
        if transaction.environment.blob_hashes.is_empty() {
            return Err("BlobTransactionMissingBlobHashes");
        }
        if transaction.to.is_none() {
            return Err("BlobTransactionContractCreation");
        }
        // Prague's EIP-7691 raises the block/transaction allowance to nine;
        // Osaka's EIP-7934 caps a single blob transaction back at six.
        let max_blobs = if transaction.fork == Fork::Prague {
            9
        } else {
            6
        };
        if transaction.fork != Fork::Cancun && transaction.environment.blob_hashes.len() > 9 {
            return Err("BlobGasAllowanceExceeded");
        }
        if transaction.environment.blob_hashes.len() > max_blobs {
            return Err("TooManyBlobs");
        }
        if transaction
            .environment
            .blob_hashes
            .iter()
            .any(|hash| hash[0] != 1)
        {
            return Err("InvalidBlobVersionedHash");
        }
    } else if !transaction.environment.blob_hashes.is_empty() {
        return Err("UnexpectedBlobHashes");
    }
    if transaction.max_fee_per_gas < transaction.environment.base_fee {
        return Err("MaxFeePerGasBelowBaseFee");
    }
    if transaction
        .max_priority_fee_per_gas
        .is_some_and(|priority| priority > transaction.max_fee_per_gas)
    {
        return Err("PriorityFeeAboveMaxFee");
    }
    let sender = state
        .account(transaction.caller)
        .cloned()
        .unwrap_or_default();
    if !sender.code.is_empty() && delegation_target(&sender.code, transaction.fork).is_none() {
        return Err("SenderNotExternallyOwned");
    }
    if sender.nonce != transaction.nonce {
        return Err("NonceMismatch");
    }
    if sender.nonce == u64::MAX {
        return Err("NonceOverflow");
    }
    let intrinsic = intrinsic_gas(
        &transaction.data,
        transaction.to.is_none(),
        &transaction.access_list,
        transaction.authorization_list.len(),
    );
    let calldata_floor = calldata_floor_gas(&transaction.data);
    if transaction.gas_limit < intrinsic {
        return Err("IntrinsicGasTooLow");
    }
    if transaction.fork != Fork::Cancun && transaction.gas_limit < calldata_floor {
        return Err("CalldataFloorGasTooLow");
    }
    let gas_cost = transaction
        .gas_price
        .checked_mul(U256::from(transaction.gas_limit))
        .ok_or("GasPaymentOverflow")?;
    let blob_gas = U256::from(transaction.environment.blob_hashes.len() * 131_072);
    let blob_cost = blob_gas
        .checked_mul(transaction.environment.blob_base_fee)
        .ok_or("BlobGasPaymentOverflow")?;
    if transaction
        .max_fee_per_blob_gas
        .is_some_and(|fee| fee < transaction.environment.blob_base_fee)
    {
        return Err("BlobGasPriceTooLow");
    }
    let validation_gas_cost = transaction
        .max_fee_per_gas
        .checked_mul(U256::from(transaction.gas_limit))
        .ok_or("GasPaymentOverflow")?;
    let validation_blob_cost = blob_gas
        .checked_mul(transaction.max_fee_per_blob_gas.unwrap_or_default())
        .ok_or("BlobGasPaymentOverflow")?;
    let required_balance = validation_gas_cost
        .checked_add(validation_blob_cost)
        .and_then(|cost| cost.checked_add(transaction.value))
        .ok_or("UpfrontCostOverflow")?;
    let upfront = gas_cost
        .checked_add(transaction.value)
        .and_then(|value| value.checked_add(blob_cost))
        .ok_or("UpfrontCostOverflow")?;
    if sender.balance < required_balance || sender.balance < upfront {
        return Err("InsufficientFunds");
    }

    state.begin_transaction();
    warm_precompiles(state, transaction.fork);
    state.account_mut(transaction.caller).balance -= gas_cost;
    state.account_mut(transaction.caller).balance -= blob_cost;
    state.account_mut(transaction.caller).nonce += 1;
    apply_authorizations(state, &transaction);
    let snapshot = state.clone();
    let target = transaction
        .to
        .unwrap_or_else(|| create_address(transaction.caller, transaction.nonce));
    let create_collision = transaction.to.is_none()
        && state.account(target).is_some_and(|account| {
            account.nonce != 0 || !account.code.is_empty() || !account.storage.is_empty()
        });
    if !create_collision && !state.transfer(transaction.caller, target, transaction.value) {
        return Err("InsufficientFunds");
    }
    if transaction.to.is_none() && !create_collision {
        state.account_mut(target).nonce = 1;
        state.created.insert(target);
    }
    state.warm_addresses.insert(transaction.caller);
    state.warm_addresses.insert(target);
    state
        .warm_addresses
        .insert(transaction.environment.coinbase);
    for (address, slots) in &transaction.access_list {
        state.warm_addresses.insert(*address);
        for slot in slots {
            state.warm_slots.insert((*address, *slot));
        }
    }

    let available = transaction.gas_limit - intrinsic;
    let (mut halt, mut remaining, steps) = if create_collision {
        (Halt::Fault("CreateCollision"), 0, Vec::new())
    } else if transaction.to.is_some() && is_precompile(target, transaction.fork) {
        let (halt, remaining) =
            run_precompile(target, transaction.data, available, transaction.fork);
        (halt, remaining, Vec::new())
    } else {
        let code = if transaction.to.is_some() {
            if let Some(delegate) = delegation_target(state.code(target), transaction.fork) {
                state.warm_addresses.insert(delegate);
                if is_precompile(delegate, transaction.fork) {
                    Vec::new()
                } else {
                    state.code(delegate).to_vec()
                }
            } else {
                state.code(target).to_vec()
            }
        } else {
            transaction.data.clone()
        };
        let calldata = if transaction.to.is_some() {
            transaction.data
        } else {
            Vec::new()
        };
        let mut machine = Machine::new_frame(
            code,
            calldata,
            available,
            transaction.fork,
            transaction.trace,
            state,
            target,
            transaction.caller,
            transaction.caller,
            transaction.value,
            0,
            false,
            transaction.gas_price,
            transaction.environment.clone(),
        );
        let halt = machine.run();
        let remaining = machine.gas;
        let steps = std::mem::take(&mut machine.steps);
        (halt, remaining, steps)
    };
    if transaction.to.is_none()
        && let Halt::Return(code) = &halt
    {
        let deposit = code.len() as u64 * 200;
        if code.len() > 24_576 || code.first() == Some(&0xef) || remaining < deposit {
            halt = Halt::Fault("ContractCodeDepositFailed");
            remaining = 0;
        } else {
            remaining -= deposit;
        }
    }
    if matches!(halt, Halt::Revert(_) | Halt::Fault(_)) {
        *state = snapshot;
    }
    if transaction.to.is_none()
        && let Halt::Return(code) = &halt
    {
        state.account_mut(target).code = code.clone();
        state.created.insert(target);
    }
    let used_before_refund = transaction.gas_limit.saturating_sub(remaining);
    let refund_gas = (state.refund.max(0) as u64).min(used_before_refund / 5);
    remaining = remaining.saturating_add(refund_gas);
    let mut gas_used = transaction.gas_limit.saturating_sub(remaining);
    if transaction.fork != Fork::Cancun {
        gas_used = gas_used.max(calldata_floor);
    }
    let refund = U256::from(transaction.gas_limit.saturating_sub(gas_used)) * transaction.gas_price;
    state.account_mut(transaction.caller).balance =
        state.balance(transaction.caller).wrapping_add(refund);
    let priority_fee = transaction
        .gas_price
        .saturating_sub(transaction.environment.base_fee);
    let miner_reward = U256::from(gas_used) * priority_fee;
    state.account_mut(transaction.environment.coinbase).balance = state
        .balance(transaction.environment.coinbase)
        .wrapping_add(miner_reward);
    state.finalize_transaction();
    Ok(result_from_halt(
        halt,
        transaction.gas_limit,
        transaction.gas_limit.saturating_sub(gas_used),
        steps,
        transaction.trace,
        state,
        target,
    ))
}

pub(super) fn result_from_halt(
    halt: Halt,
    gas_limit: u64,
    remaining: u64,
    steps: Vec<TraceStep>,
    trace: bool,
    state: &WorldState,
    address: Address,
) -> ExecutionResult {
    let (status, output, error, gas_used) = match halt {
        Halt::Stop => (
            ExecutionStatus::Success,
            Vec::new(),
            None,
            gas_limit.saturating_sub(remaining),
        ),
        Halt::Return(output) => (
            ExecutionStatus::Success,
            output,
            None,
            gas_limit.saturating_sub(remaining),
        ),
        Halt::Revert(output) => (
            ExecutionStatus::Revert,
            output,
            None,
            gas_limit.saturating_sub(remaining),
        ),
        Halt::Fault(error) => (
            ExecutionStatus::Fault,
            Vec::new(),
            Some(error.to_owned()),
            gas_limit.saturating_sub(remaining),
        ),
    };
    let storage = state
        .account(address)
        .into_iter()
        .flat_map(|account| &account.storage)
        .map(|(slot, value)| (format!("0x{slot:064x}"), format!("0x{value:064x}")))
        .collect();
    let logs = state
        .logs
        .iter()
        .map(|log| ExecutionLog {
            address: log.address.to_string(),
            topics: log.topics().iter().map(ToString::to_string).collect(),
            data: format!("0x{}", hex::encode(&log.data.data)),
        })
        .collect();
    ExecutionResult {
        engine: ENGINE_NAME.into(),
        engine_version: ENGINE_VERSION.into(),
        status,
        return_data: format!("0x{}", hex::encode(output)),
        gas_used,
        logs,
        logs_hash: keccak256(alloy_rlp::encode(&state.logs)).to_string(),
        state_root: state.state_root().to_string(),
        storage,
        trace: trace.then_some(steps),
        error,
    }
}

pub(super) fn intrinsic_gas(
    data: &[u8],
    create: bool,
    access_list: &[(Address, Vec<U256>)],
    authorizations: usize,
) -> u64 {
    let data_gas = data
        .iter()
        .map(|byte| if *byte == 0 { 4 } else { 16 })
        .sum::<u64>();
    let access_gas = access_list
        .iter()
        .map(|(_, slots)| 2_400 + slots.len() as u64 * 1_900)
        .sum::<u64>();
    TX_BASE_GAS
        + data_gas
        + access_gas
        + authorizations as u64 * 25_000
        + if create {
            32_000 + words(data.len()) * 2
        } else {
            0
        }
}

pub(super) fn calldata_floor_gas(data: &[u8]) -> u64 {
    TX_BASE_GAS
        + data
            .iter()
            .map(|byte| if *byte == 0 { 10 } else { 40 })
            .sum::<u64>()
}
