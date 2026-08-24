use super::*;

impl<'a> Machine<'a> {
    pub(super) fn activated(&self, op: u8) -> bool {
        match op {
            0x1e => self.fork == Fork::Osaka,
            0x4b | 0xd0..=0xef | 0xf7..=0xf9 | 0xfb => false,
            _ => true,
        }
    }

    pub(super) fn unary(&mut self, gas: u64, operation: fn(U256) -> U256) -> Result<(), Halt> {
        self.charge(gas)?;
        let value = self.pop()?;
        self.push(operation(value))
    }

    pub(super) fn binary(
        &mut self,
        gas: u64,
        operation: fn(U256, U256) -> U256,
    ) -> Result<(), Halt> {
        self.charge(gas)?;
        let a = self.pop()?;
        let b = self.pop()?;
        self.push(operation(a, b))
    }

    pub(super) fn ternary(
        &mut self,
        gas: u64,
        operation: fn(U256, U256, U256) -> U256,
    ) -> Result<(), Halt> {
        self.charge(gas)?;
        let a = self.pop()?;
        let b = self.pop()?;
        let c = self.pop()?;
        self.push(operation(a, b, c))
    }

    pub(super) fn push_immediate(&mut self, op: u8) -> Result<(), Halt> {
        self.charge(3)?;
        let width = usize::from(op - 0x5f);
        let end = (self.pc + width).min(self.code.len());
        let available = end - self.pc;
        let mut word = [0u8; 32];
        word[32 - width..32 - width + available].copy_from_slice(&self.code[self.pc..end]);
        self.pc = self.pc.saturating_add(width);
        self.push(U256::from_be_bytes(word))
    }

    pub(super) fn exp(&mut self) -> Result<(), Halt> {
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

    pub(super) fn keccak(&mut self) -> Result<(), Halt> {
        let offset = self.pop()?;
        let size = self.pop()?;
        let (offset, size) = memory_region(offset, size)?;
        self.charge(30 + 6 * words(size))?;
        self.expand(offset, size)?;
        self.push(U256::from_be_bytes(
            keccak256(&self.memory[offset..offset + size]).0,
        ))
    }

    pub(super) fn copy_data(&mut self, source: DataSource) -> Result<(), Halt> {
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

    pub(super) fn extcodecopy(&mut self) -> Result<(), Halt> {
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

    pub(super) fn mcopy(&mut self) -> Result<(), Halt> {
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

    pub(super) fn log(&mut self, op: u8) -> Result<(), Halt> {
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

    pub(super) fn dup(&mut self, op: u8) -> Result<(), Halt> {
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

    pub(super) fn swap(&mut self, op: u8) -> Result<(), Halt> {
        self.charge(3)?;
        let depth = usize::from(op - 0x8f);
        if self.stack.len() <= depth {
            return Err(Halt::Fault("StackUnderflow"));
        }
        let top = self.stack.len() - 1;
        self.stack.swap(top, top - depth);
        Ok(())
    }

    pub(super) fn mload(&mut self) -> Result<(), Halt> {
        self.charge(3)?;
        let offset = self.pop_usize()?;
        self.expand(offset, 32)?;
        let mut word = [0u8; 32];
        word.copy_from_slice(&self.memory[offset..offset + 32]);
        self.push(U256::from_be_bytes(word))
    }

    pub(super) fn mstore(&mut self) -> Result<(), Halt> {
        self.charge(3)?;
        let offset = self.pop_usize()?;
        let value = self.pop()?;
        self.expand(offset, 32)?;
        self.memory[offset..offset + 32].copy_from_slice(&value.to_be_bytes::<32>());
        Ok(())
    }

    pub(super) fn mstore8(&mut self) -> Result<(), Halt> {
        self.charge(3)?;
        let offset = self.pop_usize()?;
        let value = self.pop()?;
        self.expand(offset, 1)?;
        self.memory[offset] = value.byte(0);
        Ok(())
    }

    pub(super) fn jump(&mut self, conditional: bool) -> Result<(), Halt> {
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

    pub(super) fn in_push_data(&self, destination: usize) -> bool {
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

    pub(super) fn copy_return_data(&mut self) -> Result<(), Halt> {
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
}
