use super::*;

impl<'a> Machine<'a> {
    pub(super) fn call(&mut self, op: u8) -> Result<(), Halt> {
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

    pub(super) fn create(&mut self, op: u8) -> Result<(), Halt> {
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
    pub(super) fn new_frame<'b>(
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

    pub(super) fn append_child_steps(&mut self, mut steps: Vec<TraceStep>) {
        let base = self.steps.len();
        for (offset, step) in steps.iter_mut().enumerate() {
            step.index = base + offset;
        }
        self.steps.extend(steps);
    }
}
