//! EchoEVM's independent legacy-bytecode interpreter.

use crate::{DEFAULT_GAS_LIMIT, ENGINE_NAME, ENGINE_VERSION, Fork, opcode, state::WorldState};
use alloy_primitives::{Address, B256, Bytes, Log, U256, keccak256};
use echoevm_protocol::{ExecutionLog, ExecutionResult, ExecutionStatus, TraceStep};
use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
use num_bigint::BigUint;
use p256::ecdsa::{
    Signature as P256Signature, VerifyingKey as P256VerifyingKey,
    signature::hazmat::PrehashVerifier,
};
use ripemd::Ripemd160;
use sha2::{Digest, Sha256};

use crate::kzg::verify_point_evaluation;
use std::collections::BTreeMap;

const TX_BASE_GAS: u64 = 21_000;
const OSAKA_TRANSACTION_GAS_LIMIT: u64 = 1 << 24;
const STACK_LIMIT: usize = 1_024;
#[cfg(not(target_family = "wasm"))]
const EVM_HOST_STACK_SIZE: usize = 64 * 1024 * 1024;

#[derive(Clone, Debug)]
pub struct Request {
    pub bytecode: Vec<u8>,
    pub calldata: Vec<u8>,
    pub gas_limit: u64,
    pub fork: Fork,
    pub trace: bool,
}

#[derive(Clone, Debug)]
pub struct Environment {
    pub chain_id: u64,
    pub block_number: u64,
    pub timestamp: u64,
    pub coinbase: Address,
    pub block_gas_limit: u64,
    pub base_fee: U256,
    pub prevrandao: U256,
    pub blob_base_fee: U256,
    pub block_hashes: BTreeMap<u64, B256>,
    pub blob_hashes: Vec<B256>,
}

impl Default for Environment {
    fn default() -> Self {
        Self {
            chain_id: 1,
            block_number: 0,
            timestamp: 0,
            coinbase: Address::ZERO,
            block_gas_limit: DEFAULT_GAS_LIMIT,
            base_fee: U256::ZERO,
            prevrandao: U256::ZERO,
            blob_base_fee: U256::ZERO,
            block_hashes: BTreeMap::new(),
            blob_hashes: Vec::new(),
        }
    }
}

#[derive(Clone, Debug)]
pub struct Transaction {
    pub caller: Address,
    pub to: Option<Address>,
    pub value: U256,
    pub data: Vec<u8>,
    pub gas_limit: u64,
    pub gas_price: U256,
    pub max_fee_per_gas: U256,
    pub max_priority_fee_per_gas: Option<U256>,
    pub nonce: u64,
    pub access_list: Vec<(Address, Vec<U256>)>,
    pub authorization_list: Vec<Authorization>,
    pub set_code: bool,
    pub max_fee_per_blob_gas: Option<U256>,
    pub fork: Fork,
    pub environment: Environment,
    pub trace: bool,
}

#[derive(Clone, Debug)]
pub struct Authorization {
    pub chain_id: U256,
    pub delegate: Address,
    pub nonce: u64,
    pub y_parity: u8,
    pub r: U256,
    pub s: U256,
}

#[derive(Clone, Debug, PartialEq, Eq)]
enum Halt {
    Stop,
    Return(Vec<u8>),
    Revert(Vec<u8>),
    Fault(&'static str),
}

pub fn execute(request: Request) -> ExecutionResult {
    with_evm_stack(move || execute_inner(request))
}

fn execute_inner(request: Request) -> ExecutionResult {
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

fn deploy_inner(initcode: Vec<u8>, gas_limit: u64, fork: Fork, trace: bool) -> ExecutionResult {
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

fn deploy_and_call_inner(
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

fn transact_inner(
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

#[cfg(not(target_family = "wasm"))]
fn with_evm_stack<R, F>(operation: F) -> R
where
    R: Send,
    F: FnOnce() -> R + Send,
{
    std::thread::scope(|scope| {
        let handle = std::thread::Builder::new()
            .name("echoevm-execution".into())
            .stack_size(EVM_HOST_STACK_SIZE)
            .spawn_scoped(scope, operation)
            .expect("failed to create EchoEVM execution thread");
        match handle.join() {
            Ok(result) => result,
            Err(panic) => std::panic::resume_unwind(panic),
        }
    })
}

#[cfg(target_family = "wasm")]
fn with_evm_stack<R, F>(operation: F) -> R
where
    F: FnOnce() -> R,
{
    operation()
}

fn result_from_halt(
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

struct Machine<'a> {
    code: Vec<u8>,
    calldata: Vec<u8>,
    pc: usize,
    stack: Vec<U256>,
    memory: Vec<u8>,
    return_data: Vec<u8>,
    state: &'a mut WorldState,
    address: Address,
    caller: Address,
    origin: Address,
    call_value: U256,
    gas_price: U256,
    environment: Environment,
    depth: usize,
    static_mode: bool,
    gas: u64,
    fork: Fork,
    trace: bool,
    steps: Vec<TraceStep>,
}

impl<'a> Machine<'a> {
    fn new(request: Request, state: &'a mut WorldState, address: Address) -> Self {
        Self {
            code: request.bytecode,
            calldata: request.calldata,
            pc: 0,
            stack: Vec::new(),
            memory: Vec::new(),
            return_data: Vec::new(),
            state,
            address,
            caller: Address::ZERO,
            origin: Address::ZERO,
            call_value: U256::ZERO,
            gas_price: U256::ZERO,
            environment: Environment::default(),
            depth: 0,
            static_mode: false,
            gas: request.gas_limit.saturating_sub(TX_BASE_GAS),
            fork: request.fork,
            trace: request.trace,
            steps: Vec::new(),
        }
    }

    fn run(&mut self) -> Halt {
        loop {
            if self.pc >= self.code.len() {
                return Halt::Stop;
            }
            let pc = self.pc;
            let op = self.code[self.pc];
            self.pc += 1;
            let name = opcode::name(op).unwrap_or("UNKNOWN");
            let gas_before = self.gas;
            let trace_index = self.trace.then(|| {
                let stack_before = self.stack_snapshot();
                let index = self.steps.len();
                self.steps.push(TraceStep {
                    index,
                    depth: self.depth,
                    address: Some(self.address.to_string()),
                    pc: pc as u64,
                    opcode: format!("0x{op:02x}"),
                    opcode_name: name.into(),
                    gas_before,
                    gas_after: gas_before,
                    stack_before,
                    stack_after: None,
                    halt_class: None,
                });
                index
            });
            let result = self.step(op, pc);
            if matches!(result, Err(Halt::Fault(_))) {
                self.gas = 0;
            }
            if let Some(index) = trace_index {
                let stack_after = self.stack_snapshot();
                let step = &mut self.steps[index];
                step.gas_after = self.gas;
                step.stack_after = Some(stack_after);
                step.halt_class = result.as_ref().err().map(halt_name).map(str::to_owned);
            }
            if let Err(halt) = result {
                return halt;
            }
        }
    }

    fn step(&mut self, op: u8, instruction_pc: usize) -> Result<(), Halt> {
        if !self.activated(op) {
            return Err(Halt::Fault("NotActivated"));
        }
        match op {
            0x00 => Err(Halt::Stop),
            0x01 => self.binary(3, U256::wrapping_add),
            0x02 => self.binary(5, U256::wrapping_mul),
            0x03 => self.binary(3, U256::wrapping_sub),
            0x04 => self.binary(5, |a, b| if b.is_zero() { U256::ZERO } else { a / b }),
            0x05 => self.binary(5, signed_div),
            0x06 => self.binary(5, |a, b| if b.is_zero() { U256::ZERO } else { a % b }),
            0x07 => self.binary(5, signed_mod),
            0x08 => self.ternary(8, |a, b, n| {
                if n.is_zero() {
                    U256::ZERO
                } else {
                    a.add_mod(b, n)
                }
            }),
            0x09 => self.ternary(8, |a, b, n| {
                if n.is_zero() {
                    U256::ZERO
                } else {
                    a.mul_mod(b, n)
                }
            }),
            0x0a => self.exp(),
            0x0b => self.binary(5, sign_extend),
            0x10 => self.binary(3, |a, b| U256::from(a < b)),
            0x11 => self.binary(3, |a, b| U256::from(a > b)),
            0x12 => self.binary(3, |a, b| U256::from(signed_lt(a, b))),
            0x13 => self.binary(3, |a, b| U256::from(signed_lt(b, a))),
            0x14 => self.binary(3, |a, b| U256::from(a == b)),
            0x15 => self.unary(3, |a| U256::from(a.is_zero())),
            0x16 => self.binary(3, |a, b| a & b),
            0x17 => self.binary(3, |a, b| a | b),
            0x18 => self.binary(3, |a, b| a ^ b),
            0x19 => self.unary(3, |a| !a),
            0x1a => self.binary(3, |index, value| {
                if index >= U256::from(32) {
                    U256::ZERO
                } else {
                    let shift = (31 - index.to::<usize>()) * 8;
                    (value >> shift) & U256::from(0xff)
                }
            }),
            0x1b => self.binary(3, |shift, value| {
                if shift >= U256::from(256) {
                    U256::ZERO
                } else {
                    value << shift.to::<usize>()
                }
            }),
            0x1c => self.binary(3, |shift, value| {
                if shift >= U256::from(256) {
                    U256::ZERO
                } else {
                    value >> shift.to::<usize>()
                }
            }),
            0x1d => self.binary(3, arithmetic_shift_right),
            0x1e => self.unary(5, |value| U256::from(value.leading_zeros())),
            0x20 => self.keccak(),
            0x30 => {
                self.charge(2)?;
                self.push(address_word(self.address))
            }
            0x32 => {
                self.charge(2)?;
                self.push(address_word(self.origin))
            }
            0x33 => {
                self.charge(2)?;
                self.push(address_word(self.caller))
            }
            0x34 => {
                self.charge(2)?;
                self.push(self.call_value)
            }
            0x3a => {
                self.charge(2)?;
                self.push(self.gas_price)
            }
            0x40 => {
                self.charge(20)?;
                let number = self.pop_u64_saturated()?;
                let in_range = number < self.environment.block_number
                    && self.environment.block_number - number <= 256;
                let hash = in_range
                    .then(|| self.environment.block_hashes.get(&number).copied())
                    .flatten()
                    .map(|hash| U256::from_be_bytes(hash.0))
                    .unwrap_or_default();
                self.push(hash)
            }
            0x41 => {
                self.charge(2)?;
                self.push(address_word(self.environment.coinbase))
            }
            0x42 => {
                self.charge(2)?;
                self.push(U256::from(self.environment.timestamp))
            }
            0x43 => {
                self.charge(2)?;
                self.push(U256::from(self.environment.block_number))
            }
            0x44 => {
                self.charge(2)?;
                self.push(self.environment.prevrandao)
            }
            0x45 => {
                self.charge(2)?;
                self.push(U256::from(self.environment.block_gas_limit))
            }
            0x46 => {
                self.charge(2)?;
                self.push(U256::from(self.environment.chain_id))
            }
            0x47 => {
                self.charge(5)?;
                self.push(self.state.balance(self.address))
            }
            0x48 => {
                self.charge(2)?;
                self.push(self.environment.base_fee)
            }
            0x49 => {
                self.charge(3)?;
                let index = self.pop()?;
                let value = if index > U256::from(usize::MAX) {
                    U256::ZERO
                } else {
                    self.environment
                        .blob_hashes
                        .get(index.to::<usize>())
                        .map(|hash| U256::from_be_bytes(hash.0))
                        .unwrap_or_default()
                };
                self.push(value)
            }
            0x4a => {
                self.charge(environment_gas(op))?;
                self.push(self.environment.blob_base_fee)
            }
            0x31 => {
                let address = word_address(self.pop()?);
                let cold =
                    !is_precompile(address, self.fork) && self.state.warm_addresses.insert(address);
                self.charge(if cold { 2_600 } else { 100 })?;
                self.push(self.state.balance(address))
            }
            0x3b => {
                let address = word_address(self.pop()?);
                let cold =
                    !is_precompile(address, self.fork) && self.state.warm_addresses.insert(address);
                self.charge(if cold { 2_600 } else { 100 })?;
                self.push(U256::from(self.state.code(address).len()))
            }
            0x3f => {
                let address = word_address(self.pop()?);
                let cold =
                    !is_precompile(address, self.fork) && self.state.warm_addresses.insert(address);
                self.charge(if cold { 2_600 } else { 100 })?;
                let value = self
                    .state
                    .account(address)
                    .map(|account| U256::from_be_bytes(account.code_hash().0))
                    .unwrap_or_default();
                self.push(value)
            }
            0x35 => {
                self.charge(3)?;
                let offset = self.pop()?;
                let mut word = [0u8; 32];
                if offset <= U256::from(usize::MAX) {
                    let offset = offset.to::<usize>();
                    if offset < self.calldata.len() {
                        let size = (self.calldata.len() - offset).min(32);
                        word[..size].copy_from_slice(&self.calldata[offset..offset + size]);
                    }
                }
                self.push(U256::from_be_bytes(word))
            }
            0x36 => {
                self.charge(2)?;
                self.push(U256::from(self.calldata.len()))
            }
            0x38 => {
                self.charge(2)?;
                self.push(U256::from(self.code.len()))
            }
            0x37 => self.copy_data(DataSource::Calldata),
            0x39 => self.copy_data(DataSource::Code),
            0x3c => self.extcodecopy(),
            0x3d => {
                self.charge(2)?;
                self.push(U256::from(self.return_data.len()))
            }
            0x3e => self.copy_return_data(),
            0x4b => Err(Halt::Fault("NotActivated")),
            0x50 => {
                self.charge(2)?;
                self.pop().map(|_| ())
            }
            0x51 => self.mload(),
            0x52 => self.mstore(),
            0x53 => self.mstore8(),
            0x54 => {
                let key = self.pop()?;
                let cold = self.state.warm_slots.insert((self.address, key));
                self.charge(if cold { 2_100 } else { 100 })?;
                self.push(self.state.storage(self.address, key))
            }
            0x55 => {
                if self.static_mode {
                    return Err(Halt::Fault("StateChangeDuringStaticCall"));
                }
                // EIP-2200's sentry prevents SSTORE when at most the CALL
                // stipend remains, even when the eventual write would be a
                // warm no-op costing less than 2,300 gas.
                if self.gas <= 2_300 {
                    return Err(Halt::Fault("OutOfGas"));
                }
                let key = self.pop()?;
                let value = self.pop()?;
                let current = self.state.storage(self.address, key);
                let original = self
                    .state
                    .original_storage
                    .get(&(self.address, key))
                    .copied()
                    .unwrap_or_default();
                let cold_cost = if self.state.warm_slots.insert((self.address, key)) {
                    2_100
                } else {
                    0
                };
                let gas = if current == value {
                    100
                } else if original == current {
                    if original.is_zero() {
                        20_000
                    } else {
                        if value.is_zero() {
                            self.state.refund += 4_800;
                        }
                        2_900
                    }
                } else {
                    if !original.is_zero() {
                        if current.is_zero() {
                            self.state.refund -= 4_800;
                        }
                        if value.is_zero() {
                            self.state.refund += 4_800;
                        }
                    }
                    if value == original {
                        self.state.refund += if original.is_zero() { 19_900 } else { 2_800 };
                    }
                    100
                };
                self.charge(gas + cold_cost)?;
                if value.is_zero() {
                    self.state.set_storage(self.address, key, U256::ZERO);
                } else {
                    self.state.set_storage(self.address, key, value);
                }
                Ok(())
            }
            0x56 => self.jump(false),
            0x57 => self.jump(true),
            0x58 => {
                self.charge(2)?;
                self.push(U256::from(instruction_pc))
            }
            0x59 => {
                self.charge(2)?;
                self.push(U256::from(self.memory.len()))
            }
            0x5a => {
                self.charge(2)?;
                self.push(U256::from(self.gas))
            }
            0x5b => self.charge(1),
            0x5c => {
                self.charge(100)?;
                let key = self.pop()?;
                self.push(
                    self.state
                        .transient
                        .get(&(self.address, key))
                        .copied()
                        .unwrap_or_default(),
                )
            }
            0x5d => {
                if self.static_mode {
                    return Err(Halt::Fault("StateChangeDuringStaticCall"));
                }
                self.charge(100)?;
                let key = self.pop()?;
                let value = self.pop()?;
                self.state.transient.insert((self.address, key), value);
                Ok(())
            }
            0x5e => self.mcopy(),
            0x5f => {
                self.charge(2)?;
                self.push(U256::ZERO)
            }
            0x60..=0x7f => self.push_immediate(op),
            0x80..=0x8f => self.dup(op),
            0x90..=0x9f => self.swap(op),
            0xa0..=0xa4 => self.log(op),
            0xf0 | 0xf5 => self.create(op),
            0xf1 | 0xf2 | 0xf4 => self.call(op),
            0xf3 => {
                let output = self.output_region()?;
                Err(Halt::Return(output))
            }
            0xfa => self.call(op),
            0xfd => {
                let output = self.output_region()?;
                Err(Halt::Revert(output))
            }
            0xfe => Err(Halt::Fault("InvalidFEOpcode")),
            0xff => {
                if self.static_mode {
                    return Err(Halt::Fault("StateChangeDuringStaticCall"));
                }
                let beneficiary = word_address(self.pop()?);
                let cold = self.state.warm_addresses.insert(beneficiary);
                let balance = self.state.balance(self.address);
                let creates_beneficiary = beneficiary != self.address
                    && !balance.is_zero()
                    && self.state.account(beneficiary).is_none_or(|account| {
                        account.nonce == 0 && account.balance.is_zero() && account.code.is_empty()
                    });
                self.charge(
                    5_000
                        + if cold { 2_600 } else { 0 }
                        + if creates_beneficiary { 25_000 } else { 0 },
                )?;
                if beneficiary != self.address {
                    self.state.account_mut(self.address).balance = U256::ZERO;
                    self.state.account_mut(beneficiary).balance =
                        self.state.balance(beneficiary).wrapping_add(balance);
                } else if self.state.created.contains(&self.address) {
                    self.state.account_mut(self.address).balance = U256::ZERO;
                }
                if self.state.created.contains(&self.address) {
                    self.state.selfdestructed.insert(self.address);
                }
                Err(Halt::Stop)
            }
            0xd0..=0xef | 0xf7..=0xf9 | 0xfb => Err(Halt::Fault("NotActivated")),
            _ => Err(Halt::Fault("OpcodeNotFound")),
        }
    }

    fn activated(&self, op: u8) -> bool {
        match op {
            0x1e => self.fork == Fork::Osaka,
            0x4b | 0xd0..=0xef | 0xf7..=0xf9 | 0xfb => false,
            _ => true,
        }
    }

    fn unary(&mut self, gas: u64, operation: fn(U256) -> U256) -> Result<(), Halt> {
        self.charge(gas)?;
        let value = self.pop()?;
        self.push(operation(value))
    }

    fn binary(&mut self, gas: u64, operation: fn(U256, U256) -> U256) -> Result<(), Halt> {
        self.charge(gas)?;
        let a = self.pop()?;
        let b = self.pop()?;
        self.push(operation(a, b))
    }

    fn ternary(&mut self, gas: u64, operation: fn(U256, U256, U256) -> U256) -> Result<(), Halt> {
        self.charge(gas)?;
        let a = self.pop()?;
        let b = self.pop()?;
        let c = self.pop()?;
        self.push(operation(a, b, c))
    }

    fn push_immediate(&mut self, op: u8) -> Result<(), Halt> {
        self.charge(3)?;
        let width = usize::from(op - 0x5f);
        let end = (self.pc + width).min(self.code.len());
        let available = end - self.pc;
        let mut word = [0u8; 32];
        word[32 - width..32 - width + available].copy_from_slice(&self.code[self.pc..end]);
        self.pc = self.pc.saturating_add(width);
        self.push(U256::from_be_bytes(word))
    }

    fn exp(&mut self) -> Result<(), Halt> {
        let base = self.pop()?;
        let exponent = self.pop()?;
        let bytes = if exponent.is_zero() {
            0
        } else {
            (256 - exponent.leading_zeros()).div_ceil(8) as u64
        };
        self.charge(10 + 50 * bytes)?;
        self.push(wrapping_pow(base, exponent))
    }

    fn keccak(&mut self) -> Result<(), Halt> {
        let offset = self.pop()?;
        let size = self.pop()?;
        let (offset, size) = memory_region(offset, size)?;
        self.charge(30 + 6 * words(size))?;
        self.expand(offset, size)?;
        self.push(U256::from_be_bytes(
            keccak256(&self.memory[offset..offset + size]).0,
        ))
    }

    fn copy_data(&mut self, source: DataSource) -> Result<(), Halt> {
        self.charge(3)?;
        let memory_offset = self.pop()?;
        let data_offset = self.pop()?;
        let size = self.pop()?;
        let (memory_offset, size) = memory_region(memory_offset, size)?;
        self.charge(copy_gas(size))?;
        self.expand(memory_offset, size)?;
        if data_offset > U256::from(usize::MAX) {
            return Ok(());
        }
        let data_offset = data_offset.to::<usize>();
        let data: &[u8] = match source {
            DataSource::Calldata => &self.calldata,
            DataSource::Code => &self.code,
        };
        for index in 0..size {
            self.memory[memory_offset + index] = data_offset
                .checked_add(index)
                .and_then(|offset| data.get(offset))
                .copied()
                .unwrap_or(0);
        }
        Ok(())
    }

    fn extcodecopy(&mut self) -> Result<(), Halt> {
        let address = word_address(self.pop()?);
        let memory_offset = self.pop()?;
        let code_offset = self.pop()?;
        let size = self.pop()?;
        let (memory_offset, size) = memory_region(memory_offset, size)?;
        let cold = self.state.warm_addresses.insert(address);
        self.charge(if cold { 2_600 } else { 100 })?;
        self.charge(copy_gas(size))?;
        self.expand(memory_offset, size)?;
        if code_offset > U256::from(usize::MAX) {
            return Ok(());
        }
        let code_offset = code_offset.to::<usize>();
        let code = self.state.code(address);
        for index in 0..size {
            self.memory[memory_offset + index] = code_offset
                .checked_add(index)
                .and_then(|offset| code.get(offset))
                .copied()
                .unwrap_or(0);
        }
        Ok(())
    }

    fn mcopy(&mut self) -> Result<(), Halt> {
        self.charge(3)?;
        let destination = self.pop()?;
        let source = self.pop()?;
        let size = self.pop()?;
        if size.is_zero() {
            return Ok(());
        }
        let size = usize_from_word(size)?;
        self.charge(copy_gas(size))?;
        if size == 0 {
            return Ok(());
        }
        let destination = usize_from_word(destination)?;
        let source = usize_from_word(source)?;
        self.expand(destination, size)?;
        self.expand(source, size)?;
        self.memory.copy_within(source..source + size, destination);
        Ok(())
    }

    fn log(&mut self, op: u8) -> Result<(), Halt> {
        if self.static_mode {
            return Err(Halt::Fault("StateChangeDuringStaticCall"));
        }
        let topics = usize::from(op - 0xa0);
        let offset = self.pop()?;
        let size = self.pop()?;
        let (offset, size) = memory_region(offset, size)?;
        let topics = (0..topics)
            .map(|_| {
                self.pop()
                    .map(|topic| B256::from(topic.to_be_bytes::<32>()))
            })
            .collect::<Result<Vec<_>, _>>()?;
        self.charge(
            375u64
                .saturating_add(375u64.saturating_mul(topics.len() as u64))
                .saturating_add(8u64.saturating_mul(size as u64)),
        )?;
        self.expand(offset, size)?;
        self.state.logs.push(Log::new_unchecked(
            self.address,
            topics,
            Bytes::copy_from_slice(&self.memory[offset..offset + size]),
        ));
        Ok(())
    }

    fn dup(&mut self, op: u8) -> Result<(), Halt> {
        self.charge(3)?;
        let depth = usize::from(op - 0x7f);
        let value = self
            .stack
            .get(
                self.stack
                    .len()
                    .checked_sub(depth)
                    .ok_or(Halt::Fault("StackUnderflow"))?,
            )
            .copied()
            .ok_or(Halt::Fault("StackUnderflow"))?;
        self.push(value)
    }

    fn swap(&mut self, op: u8) -> Result<(), Halt> {
        self.charge(3)?;
        let depth = usize::from(op - 0x8f);
        if self.stack.len() <= depth {
            return Err(Halt::Fault("StackUnderflow"));
        }
        let top = self.stack.len() - 1;
        self.stack.swap(top, top - depth);
        Ok(())
    }

    fn mload(&mut self) -> Result<(), Halt> {
        self.charge(3)?;
        let offset = self.pop_usize()?;
        self.expand(offset, 32)?;
        let mut word = [0u8; 32];
        word.copy_from_slice(&self.memory[offset..offset + 32]);
        self.push(U256::from_be_bytes(word))
    }

    fn mstore(&mut self) -> Result<(), Halt> {
        self.charge(3)?;
        let offset = self.pop_usize()?;
        let value = self.pop()?;
        self.expand(offset, 32)?;
        self.memory[offset..offset + 32].copy_from_slice(&value.to_be_bytes::<32>());
        Ok(())
    }

    fn mstore8(&mut self) -> Result<(), Halt> {
        self.charge(3)?;
        let offset = self.pop_usize()?;
        let value = self.pop()?;
        self.expand(offset, 1)?;
        self.memory[offset] = value.byte(0);
        Ok(())
    }

    fn jump(&mut self, conditional: bool) -> Result<(), Halt> {
        self.charge(if conditional { 10 } else { 8 })?;
        let destination = self.pop()?;
        if conditional && self.pop()?.is_zero() {
            return Ok(());
        }
        if destination > U256::from(usize::MAX) {
            return Err(Halt::Fault("InvalidJump"));
        }
        let destination = destination.to::<usize>();
        if destination >= self.code.len()
            || self.code[destination] != 0x5b
            || self.in_push_data(destination)
        {
            return Err(Halt::Fault("InvalidJump"));
        }
        self.pc = destination;
        Ok(())
    }

    fn in_push_data(&self, destination: usize) -> bool {
        let mut pc = 0;
        while pc < self.code.len() {
            if pc == destination {
                return false;
            }
            let op = self.code[pc];
            pc += 1 + if (0x60..=0x7f).contains(&op) {
                usize::from(op - 0x5f)
            } else {
                0
            };
            if destination < pc {
                return true;
            }
        }
        true
    }

    fn copy_return_data(&mut self) -> Result<(), Halt> {
        self.charge(3)?;
        let memory_offset = self.pop()?;
        let data_offset = self.pop()?;
        let size = self.pop()?;
        let (memory_offset, size) = memory_region(memory_offset, size)?;
        let data_offset = if data_offset > U256::from(usize::MAX) {
            return Err(Halt::Fault("OutOfOffset"));
        } else {
            data_offset.to::<usize>()
        };
        let end = data_offset
            .checked_add(size)
            .ok_or(Halt::Fault("OutOfOffset"))?;
        if end > self.return_data.len() {
            return Err(Halt::Fault("OutOfOffset"));
        }
        self.charge(copy_gas(size))?;
        if size == 0 {
            return Ok(());
        }
        self.expand(memory_offset, size)?;
        self.memory[memory_offset..memory_offset + size]
            .copy_from_slice(&self.return_data[data_offset..end]);
        Ok(())
    }

    fn call(&mut self, op: u8) -> Result<(), Halt> {
        let requested_gas = self.pop_u64_saturated()?;
        let code_address = word_address(self.pop()?);
        let value = if matches!(op, 0xf1 | 0xf2) {
            self.pop()?
        } else {
            U256::ZERO
        };
        let input_offset_word = self.pop()?;
        let input_size_word = self.pop()?;
        let output_offset_word = self.pop()?;
        let output_size_word = self.pop()?;
        let (input_offset, input_size) = memory_region(input_offset_word, input_size_word)?;
        let (output_offset, output_size) = memory_region(output_offset_word, output_size_word)?;
        if self.static_mode && op == 0xf1 && !value.is_zero() {
            return Err(Halt::Fault("StateChangeDuringStaticCall"));
        }
        self.expand(input_offset, input_size)?;
        self.expand(output_offset, output_size)?;
        let cold = !is_precompile(code_address, self.fork)
            && self.state.warm_addresses.insert(code_address);
        let mut base_cost = if cold { 2_600 } else { 100 };
        let delegated = delegation_target(self.state.code(code_address), self.fork);
        if let Some(delegate) = delegated {
            let cold =
                !is_precompile(delegate, self.fork) && self.state.warm_addresses.insert(delegate);
            base_cost += if cold { 2_600 } else { 100 };
        }
        if !value.is_zero() {
            base_cost += 9_000;
            if op == 0xf1
                && self.state.account(code_address).is_none_or(|account| {
                    account.nonce == 0 && account.balance.is_zero() && account.code.is_empty()
                })
            {
                base_cost += 25_000;
            }
        }
        self.charge(base_cost)?;
        let cap = self.gas - self.gas / 64;
        let forwarded = requested_gas.min(cap);
        let child_gas = forwarded + if value.is_zero() { 0 } else { 2_300 };
        self.charge(forwarded)?;
        if self.depth >= 1_024 {
            self.gas = self.gas.saturating_add(child_gas);
            self.return_data.clear();
            return self.push(U256::ZERO);
        }

        let input = self.memory[input_offset..input_offset + input_size].to_vec();
        let context_address = if matches!(op, 0xf2 | 0xf4) {
            self.address
        } else {
            code_address
        };
        let caller = if op == 0xf4 {
            self.caller
        } else {
            self.address
        };
        let call_value = if op == 0xf4 { self.call_value } else { value };
        let static_mode = self.static_mode || op == 0xfa;
        let snapshot = self.state.clone();
        if matches!(op, 0xf1 | 0xf2) && self.state.balance(self.address) < value {
            self.gas = self.gas.saturating_add(child_gas);
            self.return_data.clear();
            return self.push(U256::ZERO);
        }
        if op == 0xf1 && !self.state.transfer(self.address, code_address, value) {
            self.gas = self.gas.saturating_add(child_gas);
            self.return_data.clear();
            return self.push(U256::ZERO);
        }

        let (halt, remaining, child_steps) = if is_precompile(code_address, self.fork) {
            let (halt, remaining) = run_precompile(code_address, input, child_gas, self.fork);
            (halt, remaining, Vec::new())
        } else {
            let code = if let Some(delegate) = delegated {
                if is_precompile(delegate, self.fork) {
                    Vec::new()
                } else {
                    self.state.code(delegate).to_vec()
                }
            } else {
                self.state.code(code_address).to_vec()
            };
            if code.is_empty() {
                (Halt::Stop, child_gas, Vec::new())
            } else {
                let mut child = Self::new_frame(
                    code,
                    input,
                    child_gas,
                    self.fork,
                    self.trace,
                    self.state,
                    context_address,
                    caller,
                    self.origin,
                    call_value,
                    self.depth + 1,
                    static_mode,
                    self.gas_price,
                    self.environment.clone(),
                );
                let halt = child.run();
                let remaining = child.gas;
                (halt, remaining, child.steps)
            }
        };
        self.gas = self.gas.saturating_add(remaining);
        self.append_child_steps(child_steps);
        let (success, output) = match halt {
            Halt::Stop => (true, Vec::new()),
            Halt::Return(output) => (true, output),
            Halt::Revert(output) => {
                *self.state = snapshot;
                (false, output)
            }
            Halt::Fault(_) => {
                *self.state = snapshot;
                (false, Vec::new())
            }
        };
        self.return_data = output;
        let copy = output_size.min(self.return_data.len());
        self.memory[output_offset..output_offset + copy].copy_from_slice(&self.return_data[..copy]);
        self.push(U256::from(success))
    }

    fn create(&mut self, op: u8) -> Result<(), Halt> {
        if self.static_mode {
            return Err(Halt::Fault("StateChangeDuringStaticCall"));
        }
        let value = self.pop()?;
        let offset = self.pop()?;
        let size = self.pop()?;
        let (offset, size) = memory_region(offset, size)?;
        let salt = if op == 0xf5 { Some(self.pop()?) } else { None };
        self.charge(32_000 + words(size) * if op == 0xf5 { 8 } else { 2 })?;
        self.expand(offset, size)?;
        if size > 49_152 {
            return Err(Halt::Fault("CreateInitCodeSizeLimit"));
        }
        if self.depth >= 1_024 || self.state.balance(self.address) < value {
            self.return_data.clear();
            return self.push(U256::ZERO);
        }
        let initcode = self.memory[offset..offset + size].to_vec();
        let creator_nonce = self
            .state
            .account(self.address)
            .map(|a| a.nonce)
            .unwrap_or_default();
        // EIP-2681 makes an account at the nonce limit unable to create any
        // further contracts. This applies to both CREATE and CREATE2 even
        // though CREATE2 does not derive its destination from the nonce.
        if creator_nonce == u64::MAX {
            self.return_data.clear();
            return self.push(U256::ZERO);
        }
        let address = if let Some(salt) = salt {
            create2_address(self.address, salt, &initcode)
        } else {
            create_address(self.address, creator_nonce)
        };
        self.state.account_mut(self.address).nonce = creator_nonce.saturating_add(1);
        self.state.warm_addresses.insert(address);
        let forwarded = self.gas - self.gas / 64;
        self.charge(forwarded)?;
        if self.state.account(address).is_some_and(|account| {
            account.nonce != 0 || !account.code.is_empty() || !account.storage.is_empty()
        }) {
            self.return_data.clear();
            return self.push(U256::ZERO);
        }
        let snapshot = self.state.clone();
        self.state.account_mut(address).nonce = 1;
        self.state.created.insert(address);
        if !self.state.transfer(self.address, address, value) {
            *self.state = snapshot;
            return self.push(U256::ZERO);
        }
        let mut child = Self::new_frame(
            initcode,
            Vec::new(),
            forwarded,
            self.fork,
            self.trace,
            self.state,
            address,
            self.address,
            self.origin,
            value,
            self.depth + 1,
            false,
            self.gas_price,
            self.environment.clone(),
        );
        let halt = child.run();
        let mut remaining = child.gas;
        let child_steps = child.steps;
        self.append_child_steps(child_steps);
        match halt {
            Halt::Revert(output) => {
                *self.state = snapshot;
                self.gas = self.gas.saturating_add(remaining);
                self.return_data = output;
                self.push(U256::ZERO)
            }
            Halt::Fault(_) => {
                *self.state = snapshot;
                self.return_data.clear();
                self.push(U256::ZERO)
            }
            Halt::Stop | Halt::Return(_) => {
                let runtime = match halt {
                    Halt::Return(output) => output,
                    _ => Vec::new(),
                };
                let deposit = runtime.len() as u64 * 200;
                if runtime.len() > 24_576 || runtime.first() == Some(&0xef) || remaining < deposit {
                    *self.state = snapshot;
                    self.return_data.clear();
                    return self.push(U256::ZERO);
                }
                remaining -= deposit;
                self.gas = self.gas.saturating_add(remaining);
                self.state.account_mut(address).code = runtime;
                self.return_data.clear();
                self.push(address_word(address))
            }
        }
    }

    #[allow(clippy::too_many_arguments)]
    fn new_frame<'b>(
        code: Vec<u8>,
        calldata: Vec<u8>,
        gas: u64,
        fork: Fork,
        trace: bool,
        state: &'b mut WorldState,
        address: Address,
        caller: Address,
        origin: Address,
        call_value: U256,
        depth: usize,
        static_mode: bool,
        gas_price: U256,
        environment: Environment,
    ) -> Machine<'b> {
        Machine {
            code,
            calldata,
            pc: 0,
            stack: Vec::new(),
            memory: Vec::new(),
            return_data: Vec::new(),
            state,
            address,
            caller,
            origin,
            call_value,
            depth,
            static_mode,
            gas_price,
            environment,
            gas,
            fork,
            trace,
            steps: Vec::new(),
        }
    }

    fn append_child_steps(&mut self, mut steps: Vec<TraceStep>) {
        let base = self.steps.len();
        for (offset, step) in steps.iter_mut().enumerate() {
            step.index = base + offset;
        }
        self.steps.extend(steps);
    }

    fn output_region(&mut self) -> Result<Vec<u8>, Halt> {
        let offset = self.pop()?;
        let size = self.pop()?;
        let (offset, size) = memory_region(offset, size)?;
        self.expand(offset, size)?;
        Ok(self.memory[offset..offset + size].to_vec())
    }

    fn expand(&mut self, offset: usize, size: usize) -> Result<(), Halt> {
        if size == 0 {
            return Ok(());
        }
        let needed = offset.checked_add(size).ok_or(Halt::Fault("OutOfGas"))?;
        let rounded = needed.checked_add(31).ok_or(Halt::Fault("OutOfGas"))? / 32 * 32;
        if rounded <= self.memory.len() {
            return Ok(());
        }
        let old_cost = memory_cost(self.memory.len());
        let new_cost = memory_cost(rounded);
        self.charge(
            new_cost
                .checked_sub(old_cost)
                .ok_or(Halt::Fault("OutOfGas"))?,
        )?;
        self.memory.resize(rounded, 0);
        Ok(())
    }

    fn charge(&mut self, amount: u64) -> Result<(), Halt> {
        self.gas = self
            .gas
            .checked_sub(amount)
            .ok_or(Halt::Fault("OutOfGas"))?;
        Ok(())
    }

    fn pop(&mut self) -> Result<U256, Halt> {
        self.stack.pop().ok_or(Halt::Fault("StackUnderflow"))
    }

    fn pop_usize(&mut self) -> Result<usize, Halt> {
        usize_from_word(self.pop()?)
    }

    fn pop_u64_saturated(&mut self) -> Result<u64, Halt> {
        let value = self.pop()?;
        Ok(if value > U256::from(u64::MAX) {
            u64::MAX
        } else {
            value.to::<u64>()
        })
    }

    fn push(&mut self, value: U256) -> Result<(), Halt> {
        if self.stack.len() >= STACK_LIMIT {
            return Err(Halt::Fault("StackOverflow"));
        }
        self.stack.push(value);
        Ok(())
    }

    fn stack_snapshot(&self) -> Vec<String> {
        self.stack
            .iter()
            .map(|value| format!("0x{value:064x}"))
            .collect()
    }
}

fn usize_from_word(value: U256) -> Result<usize, Halt> {
    if value > U256::from(usize::MAX) {
        Err(Halt::Fault("OutOfGas"))
    } else {
        Ok(value.to::<usize>())
    }
}

fn memory_region(offset: U256, size: U256) -> Result<(usize, usize), Halt> {
    let size = usize_from_word(size)?;
    if size == 0 {
        Ok((0, 0))
    } else {
        Ok((usize_from_word(offset)?, size))
    }
}

#[derive(Clone, Copy)]
enum DataSource {
    Calldata,
    Code,
}

fn is_precompile(address: Address, fork: Fork) -> bool {
    let bytes = address.as_slice();
    if fork == Fork::Osaka {
        return is_p256verify_precompile(address, fork)
            || (bytes[..19].iter().all(|byte| *byte == 0) && (1..=17).contains(&bytes[19]));
    }
    let maximum = if fork == Fork::Cancun { 10 } else { 17 };
    bytes[..19].iter().all(|byte| *byte == 0) && (1..=maximum).contains(&bytes[19])
}

fn warm_precompiles(state: &mut WorldState, fork: Fork) {
    let maximum = if fork == Fork::Cancun { 10 } else { 17 };
    for number in 1..=maximum {
        let mut bytes = [0u8; 20];
        bytes[19] = number;
        state.warm_addresses.insert(Address::from(bytes));
    }
    if fork == Fork::Osaka {
        let mut bytes = [0u8; 20];
        bytes[18..].copy_from_slice(&0x100u16.to_be_bytes());
        state.warm_addresses.insert(Address::from(bytes));
    }
}

fn is_p256verify_precompile(address: Address, fork: Fork) -> bool {
    let bytes = address.as_slice();
    fork == Fork::Osaka && bytes[..18].iter().all(|byte| *byte == 0) && bytes[18..] == [1, 0]
}

fn delegation_target(code: &[u8], fork: Fork) -> Option<Address> {
    (fork != Fork::Cancun && code.len() == 23 && code.starts_with(&[0xef, 0x01, 0x00]))
        .then(|| Address::from_slice(&code[3..]))
}

fn run_precompile(address: Address, input: Vec<u8>, gas: u64, fork: Fork) -> (Halt, u64) {
    let bytes = address.as_slice();
    let number = u16::from_be_bytes([bytes[18], bytes[19]]);
    if number == 5
        && fork == Fork::Osaka
        && [0, 32, 64]
            .into_iter()
            .any(|offset| precompile_length(&input, offset) > 1_024)
    {
        return (Halt::Fault("PrecompileError"), 0);
    }
    let cost = match number {
        1 => 3_000,
        2 => 60 + 12 * words(input.len()),
        3 => 600 + 120 * words(input.len()),
        4 => 15 + 3 * words(input.len()),
        5 => modexp_gas(&input, fork),
        6 => 150,
        7 => 6_000,
        8 => 45_000 + 34_000 * (input.len() / 192) as u64,
        9 if input.len() == 213 => u32::from_be_bytes(input[..4].try_into().unwrap()) as u64,
        10 => 50_000,
        11..=17 if fork != Fork::Cancun => crate::bls::gas(number as u8, input.len()).unwrap(),
        0x100 if fork == Fork::Osaka => 6_900,
        _ => return (Halt::Fault("PrecompileError"), 0),
    };
    match gas.checked_sub(cost) {
        Some(remaining) => {
            let output = match number {
                1 => ecrecover_precompile(&input),
                2 => Sha256::digest(input).to_vec(),
                3 => {
                    let hash = Ripemd160::digest(input);
                    let mut output = vec![0; 12];
                    output.extend_from_slice(&hash);
                    output
                }
                4 => input,
                5 => match modexp_precompile(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                6 if input.iter().all(|byte| *byte == 0) => vec![0; 64],
                6 => match crate::bn254::add(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                7 => match crate::bn254::mul(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                8 => match crate::bn254::pairing(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                9 => match blake2f_precompile(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                10 => match verify_point_evaluation(&input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                11..=17 => match crate::bls::execute(number as u8, &input) {
                    Some(output) => output,
                    None => return (Halt::Fault("PrecompileError"), 0),
                },
                0x100 => p256verify_precompile(&input),
                _ => return (Halt::Fault("PrecompileError"), 0),
            };
            (Halt::Return(output), remaining)
        }
        None => (Halt::Fault("OutOfGas"), 0),
    }
}

fn p256verify_precompile(input: &[u8]) -> Vec<u8> {
    if input.len() != 160 {
        return Vec::new();
    }
    let Ok(signature) = P256Signature::from_scalars(
        <[u8; 32]>::try_from(&input[32..64]).unwrap(),
        <[u8; 32]>::try_from(&input[64..96]).unwrap(),
    ) else {
        return Vec::new();
    };
    let mut encoded_key = [0u8; 65];
    encoded_key[0] = 4;
    encoded_key[1..].copy_from_slice(&input[96..160]);
    let Ok(key) = P256VerifyingKey::from_sec1_bytes(&encoded_key) else {
        return Vec::new();
    };
    if key.verify_prehash(&input[..32], &signature).is_err() {
        return Vec::new();
    }
    let mut output = vec![0; 32];
    output[31] = 1;
    output
}

fn blake2f_precompile(input: &[u8]) -> Option<Vec<u8>> {
    const IV: [u64; 8] = [
        0x6a09e667f3bcc908,
        0xbb67ae8584caa73b,
        0x3c6ef372fe94f82b,
        0xa54ff53a5f1d36f1,
        0x510e527fade682d1,
        0x9b05688c2b3e6c1f,
        0x1f83d9abfb41bd6b,
        0x5be0cd19137e2179,
    ];
    const SIGMA: [[usize; 16]; 10] = [
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
        [14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3],
        [11, 8, 12, 0, 5, 2, 15, 13, 10, 14, 3, 6, 7, 1, 9, 4],
        [7, 9, 3, 1, 13, 12, 11, 14, 2, 6, 5, 10, 4, 0, 15, 8],
        [9, 0, 5, 7, 2, 4, 10, 15, 14, 1, 11, 12, 6, 8, 3, 13],
        [2, 12, 6, 10, 0, 11, 8, 3, 4, 13, 7, 5, 15, 14, 1, 9],
        [12, 5, 1, 15, 14, 13, 4, 10, 0, 7, 6, 3, 9, 2, 8, 11],
        [13, 11, 7, 14, 12, 1, 3, 9, 5, 0, 15, 4, 8, 6, 2, 10],
        [6, 15, 14, 9, 11, 3, 0, 8, 12, 2, 13, 7, 1, 4, 10, 5],
        [10, 2, 8, 4, 7, 6, 1, 5, 15, 11, 9, 14, 3, 12, 13, 0],
    ];
    if input.len() != 213 || input[212] > 1 {
        return None;
    }
    let rounds = u32::from_be_bytes(input[..4].try_into().ok()?) as usize;
    let mut h = [0u64; 8];
    let mut m = [0u64; 16];
    for (index, word) in h.iter_mut().enumerate() {
        *word = u64::from_le_bytes(input[4 + index * 8..12 + index * 8].try_into().ok()?);
    }
    for (index, word) in m.iter_mut().enumerate() {
        *word = u64::from_le_bytes(input[68 + index * 8..76 + index * 8].try_into().ok()?);
    }
    let t0 = u64::from_le_bytes(input[196..204].try_into().ok()?);
    let t1 = u64::from_le_bytes(input[204..212].try_into().ok()?);
    let mut v = [0u64; 16];
    v[..8].copy_from_slice(&h);
    v[8..].copy_from_slice(&IV);
    v[12] ^= t0;
    v[13] ^= t1;
    if input[212] == 1 {
        v[14] = !v[14];
    }
    for round in 0..rounds {
        let s = SIGMA[round % 10];
        blake2f_g(&mut v, 0, 4, 8, 12, m[s[0]], m[s[1]]);
        blake2f_g(&mut v, 1, 5, 9, 13, m[s[2]], m[s[3]]);
        blake2f_g(&mut v, 2, 6, 10, 14, m[s[4]], m[s[5]]);
        blake2f_g(&mut v, 3, 7, 11, 15, m[s[6]], m[s[7]]);
        blake2f_g(&mut v, 0, 5, 10, 15, m[s[8]], m[s[9]]);
        blake2f_g(&mut v, 1, 6, 11, 12, m[s[10]], m[s[11]]);
        blake2f_g(&mut v, 2, 7, 8, 13, m[s[12]], m[s[13]]);
        blake2f_g(&mut v, 3, 4, 9, 14, m[s[14]], m[s[15]]);
    }
    let mut output = Vec::with_capacity(64);
    for index in 0..8 {
        output.extend_from_slice(&(h[index] ^ v[index] ^ v[index + 8]).to_le_bytes());
    }
    Some(output)
}

fn blake2f_g(v: &mut [u64; 16], a: usize, b: usize, c: usize, d: usize, x: u64, y: u64) {
    v[a] = v[a].wrapping_add(v[b]).wrapping_add(x);
    v[d] = (v[d] ^ v[a]).rotate_right(32);
    v[c] = v[c].wrapping_add(v[d]);
    v[b] = (v[b] ^ v[c]).rotate_right(24);
    v[a] = v[a].wrapping_add(v[b]).wrapping_add(y);
    v[d] = (v[d] ^ v[a]).rotate_right(16);
    v[c] = v[c].wrapping_add(v[d]);
    v[b] = (v[b] ^ v[c]).rotate_right(63);
}

fn ecrecover_precompile(input: &[u8]) -> Vec<u8> {
    let mut padded = [0u8; 128];
    let copy = input.len().min(128);
    padded[..copy].copy_from_slice(&input[..copy]);
    if padded[32..63].iter().any(|byte| *byte != 0) || !matches!(padded[63], 27 | 28) {
        return Vec::new();
    }
    let Ok(mut signature) = Signature::from_scalars(
        <[u8; 32]>::try_from(&padded[64..96]).unwrap(),
        <[u8; 32]>::try_from(&padded[96..128]).unwrap(),
    ) else {
        return Vec::new();
    };
    let Some(mut recovery) = RecoveryId::from_byte(padded[63] - 27) else {
        return Vec::new();
    };
    // Ethereum's precompile accepts both low-s and high-s signatures. k256
    // verifies only low-s signatures, so preserve the recovered key by
    // normalizing s and flipping the recovery y parity together.
    if let Some(normalized) = signature.normalize_s() {
        signature = normalized;
        recovery = RecoveryId::new(!recovery.is_y_odd(), recovery.is_x_reduced());
    }
    let Ok(key) = VerifyingKey::recover_from_prehash(&padded[..32], &signature, recovery) else {
        return Vec::new();
    };
    let encoded = key.to_encoded_point(false);
    let mut output = vec![0; 12];
    output.extend_from_slice(&keccak256(&encoded.as_bytes()[1..])[12..]);
    output
}

fn modexp_gas(input: &[u8], fork: Fork) -> u64 {
    let base_len = precompile_length(input, 0);
    let exponent_len = precompile_length(input, 32);
    let modulus_len = precompile_length(input, 64);
    let max_len = base_len.max(modulus_len);
    let words = max_len.saturating_add(7) / 8;
    let complexity = if fork == Fork::Osaka {
        if max_len > 32 {
            2u64.saturating_mul(words.saturating_mul(words))
        } else {
            16
        }
    } else {
        words.saturating_mul(words)
    };
    let head_len = exponent_len.min(32) as usize;
    let base_offset = 96usize.saturating_add(base_len.min(usize::MAX as u64) as usize);
    let mut head = vec![0; head_len];
    if base_offset < input.len() {
        let copy = head_len.min(input.len() - base_offset);
        head[..copy].copy_from_slice(&input[base_offset..base_offset + copy]);
    }
    let head_bits = BigUint::from_bytes_be(&head).bits();
    let iterations = if exponent_len <= 32 {
        head_bits.saturating_sub(1)
    } else {
        (if fork == Fork::Osaka { 16u64 } else { 8u64 })
            .saturating_mul(exponent_len - 32)
            .saturating_add(head_bits.saturating_sub(1))
    }
    .max(1);
    let cost = complexity.saturating_mul(iterations);
    if fork == Fork::Osaka {
        cost.max(500)
    } else {
        cost.saturating_div(3).max(200)
    }
}

fn precompile_length(input: &[u8], offset: usize) -> u64 {
    let mut word = [0u8; 32];
    if offset < input.len() {
        let copy = 32.min(input.len() - offset);
        word[..copy].copy_from_slice(&input[offset..offset + copy]);
    }
    let value = U256::from_be_bytes(word);
    if value > U256::from(u64::MAX) {
        u64::MAX
    } else {
        value.to::<u64>()
    }
}

fn modexp_precompile(input: &[u8]) -> Option<Vec<u8>> {
    let base_len = usize::try_from(precompile_length(input, 0)).ok()?;
    let exponent_len = usize::try_from(precompile_length(input, 32)).ok()?;
    let modulus_len = usize::try_from(precompile_length(input, 64)).ok()?;
    if modulus_len == 0 {
        return Some(Vec::new());
    }
    let base_offset = 96usize;
    let exponent_offset = base_offset.checked_add(base_len)?;
    let modulus_offset = exponent_offset.checked_add(exponent_len)?;
    let base = read_padded_segment(input, base_offset, base_len)?;
    let exponent = read_padded_segment(input, exponent_offset, exponent_len)?;
    let modulus = read_padded_segment(input, modulus_offset, modulus_len)?;
    let modulus = BigUint::from_bytes_be(&modulus);
    if modulus == BigUint::from(0u8) {
        return Some(vec![0; modulus_len]);
    }
    let value = BigUint::from_bytes_be(&base)
        .modpow(&BigUint::from_bytes_be(&exponent), &modulus)
        .to_bytes_be();
    let mut output = vec![0; modulus_len];
    let copy = value.len().min(modulus_len);
    output[modulus_len - copy..].copy_from_slice(&value[value.len() - copy..]);
    Some(output)
}

fn read_padded_segment(input: &[u8], offset: usize, len: usize) -> Option<Vec<u8>> {
    let _ = offset.checked_add(len)?;
    let mut output = vec![0; len];
    if offset < input.len() {
        let copy = len.min(input.len() - offset);
        output[..copy].copy_from_slice(&input[offset..offset + copy]);
    }
    Some(output)
}

fn halt_name(halt: &Halt) -> &'static str {
    match halt {
        Halt::Stop => "Stop",
        Halt::Return(_) => "Return",
        Halt::Revert(_) => "Revert",
        Halt::Fault(reason) => reason,
    }
}

const fn words(size: usize) -> u64 {
    size.div_ceil(32) as u64
}

const fn memory_cost(size: usize) -> u64 {
    let words = words(size);
    3u64.saturating_mul(words)
        .saturating_add(words.saturating_mul(words) / 512)
}

const fn copy_gas(size: usize) -> u64 {
    3u64.saturating_mul(words(size))
}

const fn environment_gas(op: u8) -> u64 {
    match op {
        0x44 | 0x47 => 5,
        0x40 => 20,
        0x49 => 3,
        _ => 2,
    }
}

fn is_negative(value: U256) -> bool {
    value.bit(255)
}

fn twos_complement(value: U256) -> U256 {
    (!value).wrapping_add(U256::from(1))
}

fn signed_div(a: U256, b: U256) -> U256 {
    if b.is_zero() {
        return U256::ZERO;
    }
    let negative = is_negative(a) != is_negative(b);
    let left = if is_negative(a) {
        twos_complement(a)
    } else {
        a
    };
    let right = if is_negative(b) {
        twos_complement(b)
    } else {
        b
    };
    let result = left / right;
    if negative {
        twos_complement(result)
    } else {
        result
    }
}

fn signed_mod(a: U256, b: U256) -> U256 {
    if b.is_zero() {
        return U256::ZERO;
    }
    let negative = is_negative(a);
    let left = if negative { twos_complement(a) } else { a };
    let right = if is_negative(b) {
        twos_complement(b)
    } else {
        b
    };
    let result = left % right;
    if negative {
        twos_complement(result)
    } else {
        result
    }
}

fn signed_lt(a: U256, b: U256) -> bool {
    match (is_negative(a), is_negative(b)) {
        (true, false) => true,
        (false, true) => false,
        (false, false) => a < b,
        (true, true) => a < b,
    }
}

fn arithmetic_shift_right(shift: U256, value: U256) -> U256 {
    let negative = is_negative(value);
    if shift >= U256::from(256) {
        return if negative { U256::MAX } else { U256::ZERO };
    }
    let shift = shift.to::<usize>();
    if shift == 0 || !negative {
        return value >> shift;
    }
    (value >> shift) | (U256::MAX << (256 - shift))
}

fn sign_extend(byte: U256, value: U256) -> U256 {
    if byte >= U256::from(32) {
        return value;
    }
    let bit = byte.to::<usize>() * 8 + 7;
    let mask = (U256::from(1) << (bit + 1)) - U256::from(1);
    if value.bit(bit) {
        value | !mask
    } else {
        value & mask
    }
}

fn wrapping_pow(mut base: U256, mut exponent: U256) -> U256 {
    let mut result = U256::from(1);
    while !exponent.is_zero() {
        if exponent.bit(0) {
            result = result.wrapping_mul(base);
        }
        exponent >>= 1;
        base = base.wrapping_mul(base);
    }
    result
}

fn intrinsic_gas(
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

fn calldata_floor_gas(data: &[u8]) -> u64 {
    TX_BASE_GAS
        + data
            .iter()
            .map(|byte| if *byte == 0 { 10 } else { 40 })
            .sum::<u64>()
}

fn apply_authorizations(state: &mut WorldState, transaction: &Transaction) {
    for authorization in &transaction.authorization_list {
        if !(authorization.chain_id.is_zero()
            || authorization.chain_id == U256::from(transaction.environment.chain_id))
            || authorization.nonce == u64::MAX
        {
            continue;
        }
        let Some(authority) = recover_authority(authorization) else {
            continue;
        };
        state.warm_addresses.insert(authority);
        let account = state.account(authority).cloned().unwrap_or_default();
        let delegated = account.code.len() == 23 && account.code.starts_with(&[0xef, 0x01, 0x00]);
        if (!account.code.is_empty() && !delegated) || account.nonce != authorization.nonce {
            continue;
        }
        if !account.is_empty() {
            state.refund += 12_500;
        }
        let account = state.account_mut(authority);
        account.code = if authorization.delegate.is_zero() {
            Vec::new()
        } else {
            let mut code = vec![0xef, 0x01, 0x00];
            code.extend_from_slice(authorization.delegate.as_slice());
            code
        };
        account.nonce = account.nonce.saturating_add(1);
    }
}

fn recover_authority(authorization: &Authorization) -> Option<Address> {
    if authorization.y_parity > 1 {
        return None;
    }
    let signature = Signature::from_scalars(
        authorization.r.to_be_bytes::<32>(),
        authorization.s.to_be_bytes::<32>(),
    )
    .ok()?;
    if signature.normalize_s().is_some() {
        return None;
    }
    let mut payload = Vec::new();
    rlp_push_quantity(&mut payload, authorization.chain_id);
    payload.push(0x94);
    payload.extend_from_slice(authorization.delegate.as_slice());
    rlp_push_quantity(&mut payload, U256::from(authorization.nonce));
    let mut message = Vec::with_capacity(payload.len() + 2);
    message.push(0x05);
    rlp_push_list_header(&mut message, payload.len());
    message.extend(payload);
    let hash = keccak256(message);
    let key = VerifyingKey::recover_from_prehash(
        hash.as_slice(),
        &signature,
        RecoveryId::from_byte(authorization.y_parity)?,
    )
    .ok()?;
    let encoded = key.to_encoded_point(false);
    Some(Address::from_slice(
        &keccak256(&encoded.as_bytes()[1..])[12..],
    ))
}

fn rlp_push_quantity(output: &mut Vec<u8>, value: U256) {
    if value.is_zero() {
        output.push(0x80);
        return;
    }
    let bytes = value.to_be_bytes::<32>();
    let first = bytes.iter().position(|byte| *byte != 0).unwrap();
    let value = &bytes[first..];
    if value.len() == 1 && value[0] < 0x80 {
        output.push(value[0]);
    } else {
        output.push(0x80 + value.len() as u8);
        output.extend_from_slice(value);
    }
}

fn rlp_push_list_header(output: &mut Vec<u8>, len: usize) {
    if len < 56 {
        output.push(0xc0 + len as u8);
    } else {
        let bytes = len.to_be_bytes();
        let first = bytes.iter().position(|byte| *byte != 0).unwrap();
        let encoded = &bytes[first..];
        output.push(0xf7 + encoded.len() as u8);
        output.extend_from_slice(encoded);
    }
}

fn create_address(caller: Address, nonce: u64) -> Address {
    let mut payload = Vec::with_capacity(31);
    payload.push(0x94);
    payload.extend_from_slice(caller.as_slice());
    match nonce {
        0 => payload.push(0x80),
        1..=0x7f => payload.push(nonce as u8),
        _ => {
            let bytes = nonce.to_be_bytes();
            let first = bytes.iter().position(|byte| *byte != 0).unwrap_or(7);
            let encoded = &bytes[first..];
            payload.push(0x80 + encoded.len() as u8);
            payload.extend_from_slice(encoded);
        }
    }
    let mut encoded = Vec::with_capacity(payload.len() + 1);
    encoded.push(0xc0 + payload.len() as u8);
    encoded.extend(payload);
    Address::from_slice(&keccak256(encoded)[12..])
}

fn create2_address(caller: Address, salt: U256, initcode: &[u8]) -> Address {
    let mut input = Vec::with_capacity(85);
    input.push(0xff);
    input.extend_from_slice(caller.as_slice());
    input.extend_from_slice(&salt.to_be_bytes::<32>());
    input.extend_from_slice(keccak256(initcode).as_slice());
    Address::from_slice(&keccak256(input)[12..])
}

fn address_word(address: Address) -> U256 {
    U256::from_be_slice(address.as_slice())
}

fn word_address(value: U256) -> Address {
    let bytes = value.to_be_bytes::<32>();
    Address::from_slice(&bytes[12..])
}

impl Default for Request {
    fn default() -> Self {
        Self {
            bytecode: vec![0],
            calldata: Vec::new(),
            gas_limit: DEFAULT_GAS_LIMIT,
            fork: Fork::Osaka,
            trace: false,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn modexp_input(base_len: u64) -> Vec<u8> {
        let mut input = vec![0; 96];
        input[24..32].copy_from_slice(&base_len.to_be_bytes());
        input
    }

    #[test]
    fn osaka_rejects_modexp_fields_above_1024_bytes() {
        let address =
            Address::from_slice(&[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5]);
        let (halt, remaining) =
            run_precompile(address, modexp_input(1_025), 1_000_000, Fork::Osaka);
        assert_eq!(halt, Halt::Fault("PrecompileError"));
        assert_eq!(remaining, 0);
    }

    #[test]
    fn prague_retains_unbounded_modexp_behavior() {
        let address =
            Address::from_slice(&[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5]);
        let (halt, _) = run_precompile(address, modexp_input(1_025), 1_000_000, Fork::Prague);
        assert!(matches!(halt, Halt::Return(_)));
    }

    #[test]
    fn osaka_uses_eip_7883_modexp_pricing() {
        let mut input = vec![0; 96];
        input[31] = 32;
        input[63] = 32;
        input[95] = 32;
        assert_eq!(modexp_gas(&input, Fork::Prague), 200);
        assert_eq!(modexp_gas(&input, Fork::Osaka), 500);
    }

    #[test]
    fn p256verify_matches_official_valid_and_invalid_vectors() {
        let valid = hex::decode("b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9434287fa699ff2be2a4475ccd9c063d1a22a424b6ab357d9bb0b31f7a71307b9be6af716032d408183c53f6f76945363144555fad2a5ff7854159166e52fc1d0da3c553a4215893e6d95d5818df2519e13233a1e0e56e0ea7b4817a92a6973f9d8c8877a49383e20c3fdc21d4e8e280ef1feeb72c333036770369a4387168d33").unwrap();
        let invalid = hex::decode("d9eba16ed0ecae432b71fe008c98cc872bb4cc214d3220a36f365326cf807d6869cccd84ca870c08d49d596342f464017f2a05b0be539682eaa7529e4be2de362cdb85fe13cb7de39c1c7385be9f38e8bde9963ccbecd96281c4df3aca38f53782ae98a95ae76e389354f0ec660cf071309ea2d2cb14adb6543106b790be27fd77b2cdc82c3aa8f2cf21e6257c197d75f84dcd0bc2ff8875c3e245c0e0874751").unwrap();
        let mut expected = vec![0; 32];
        expected[31] = 1;
        assert_eq!(p256verify_precompile(&valid), expected);
        assert!(p256verify_precompile(&invalid).is_empty());
    }

    #[test]
    fn ecrecover_accepts_official_high_s_vector() {
        let input = hex::decode(concat!(
            "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            "000000000000000000000000000000000000000000000000000000000000001c",
            "c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5",
            "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd036413b"
        ))
        .unwrap();
        let expected =
            hex::decode("000000000000000000000000bbb10a3b5835400b63ca00372c16db781220fb0b")
                .unwrap();

        assert_eq!(ecrecover_precompile(&input), expected);
    }

    #[test]
    fn create2_rejects_creator_at_nonce_limit() {
        let creator = Address::from([0x11; 20]);
        let mut state = WorldState::default();
        state.account_mut(creator).nonce = u64::MAX;
        let mut machine = Machine::new_frame(
            vec![0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0xf5, 0x00],
            Vec::new(),
            100_000,
            Fork::Cancun,
            false,
            &mut state,
            creator,
            Address::ZERO,
            Address::ZERO,
            U256::ZERO,
            0,
            false,
            U256::ZERO,
            Environment::default(),
        );

        assert_eq!(machine.run(), Halt::Stop);
        assert_eq!(machine.stack, vec![U256::ZERO]);
        assert_eq!(machine.state.account(creator).unwrap().nonce, u64::MAX);
        assert_eq!(machine.state.accounts.len(), 1);
    }

    #[test]
    fn returndatacopy_zero_size_does_not_expand_or_index_memory() {
        let mut state = WorldState::default();
        let mut machine = Machine::new_frame(
            Vec::new(),
            Vec::new(),
            100_000,
            Fork::Cancun,
            false,
            &mut state,
            Address::ZERO,
            Address::ZERO,
            Address::ZERO,
            U256::ZERO,
            0,
            false,
            U256::ZERO,
            Environment::default(),
        );
        machine.memory.resize(32, 0);
        machine.stack = vec![U256::ZERO, U256::ZERO, U256::from(512)];

        assert_eq!(machine.copy_return_data(), Ok(()));
        assert_eq!(machine.memory.len(), 32);
    }

    #[test]
    fn zero_size_memory_region_ignores_unrepresentable_offset() {
        assert_eq!(memory_region(U256::MAX, U256::ZERO), Ok((0, 0)));
    }
}
