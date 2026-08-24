//! EchoEVM's independent legacy-bytecode interpreter.

mod address;
mod authorization;
mod control;
mod dispatch;
mod gas;
mod instructions;
mod machine;
mod math;
mod precompiles;
mod runtime;
mod transaction;

pub use transaction::{deploy, deploy_and_call, execute, transact};

use crate::{DEFAULT_GAS_LIMIT, ENGINE_NAME, ENGINE_VERSION, Fork, opcode, state::WorldState};
use alloy_primitives::{Address, B256, Bytes, Log, U256, keccak256};
use echoevm_protocol::{ExecutionLog, ExecutionResult, ExecutionStatus, TraceStep};
use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
use std::collections::BTreeMap;

use address::*;
use authorization::*;
use gas::*;
use math::*;
use precompiles::*;

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

fn halt_name(halt: &Halt) -> &'static str {
    match halt {
        Halt::Stop => "Stop",
        Halt::Return(_) => "Return",
        Halt::Revert(_) => "Revert",
        Halt::Fault(reason) => reason,
    }
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
