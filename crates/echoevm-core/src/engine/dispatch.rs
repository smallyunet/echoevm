use super::*;

impl<'a> Machine<'a> {
    pub(super) fn step(&mut self, op: u8, instruction_pc: usize) -> Result<(), Halt> {
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
}
