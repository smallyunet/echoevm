use super::*;

pub(super) fn protocol_address(value: &str) -> Address {
    value.parse().expect("hard-coded protocol address")
}

pub(super) fn system_address() -> Address {
    protocol_address("0xfffffffffffffffffffffffffffffffffffffffe")
}

pub(super) fn apply_pre_block_system_calls(
    world: &mut WorldState,
    header: &Header,
    fork: Fork,
    environment: &Environment,
) -> Result<(), ExecuteError> {
    if let Some(root) = header.parent_beacon_block_root {
        engine::system_call(
            world,
            system_address(),
            protocol_address("0x000f3df6d732807ef1319fb7b8bb8522d0beac02"),
            root.to_vec(),
            30_000_000,
            fork,
            environment.clone(),
        )
        .map_err(|error| ExecuteError::Evm(error.into()))?;
    }
    if fork != Fork::Cancun && header.number > 0 {
        engine::system_call(
            world,
            system_address(),
            protocol_address("0x0000f90827f1c53a10cb7a02335b175320002935"),
            header.parent_hash.to_vec(),
            30_000_000,
            fork,
            environment.clone(),
        )
        .map_err(|error| ExecuteError::Evm(error.into()))?;
    }
    Ok(())
}

pub(super) fn apply_post_block_system_calls(
    world: &mut WorldState,
    fork: Fork,
    environment: &Environment,
) -> Result<(), ExecuteError> {
    if fork == Fork::Cancun {
        return Ok(());
    }
    for address in [
        protocol_address("0x00000961ef480eb55e80d19ad83579a64c007002"),
        protocol_address("0x0000bbddc7ce488642fb579f8b00f3a590007251"),
    ] {
        engine::system_call(
            world,
            system_address(),
            address,
            Vec::new(),
            30_000_000,
            fork,
            environment.clone(),
        )
        .map_err(|error| ExecuteError::Evm(error.into()))?;
    }
    Ok(())
}
