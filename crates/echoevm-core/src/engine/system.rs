use super::*;

pub fn system_call(
    state: &mut WorldState,
    caller: Address,
    target: Address,
    input: Vec<u8>,
    gas_limit: u64,
    fork: Fork,
    environment: Environment,
) -> Result<Vec<u8>, &'static str> {
    with_evm_stack(move || {
        state.begin_transaction();
        warm_precompiles(state, fork);
        let snapshot = state.clone();
        let code = state.code(target).to_vec();
        if code.is_empty() {
            state.finalize_transaction();
            return Ok(Vec::new());
        }
        let mut machine = Machine::new_frame(
            code,
            input,
            gas_limit,
            fork,
            false,
            state,
            target,
            caller,
            caller,
            U256::ZERO,
            0,
            false,
            U256::ZERO,
            environment,
        );
        let halt = machine.run();
        drop(machine);
        match halt {
            Halt::Stop => {
                state.finalize_transaction();
                Ok(Vec::new())
            }
            Halt::Return(output) => {
                state.finalize_transaction();
                Ok(output)
            }
            Halt::Revert(_) | Halt::Fault(_) => {
                *state = snapshot;
                Err("SystemCallFailed")
            }
        }
    })
}
