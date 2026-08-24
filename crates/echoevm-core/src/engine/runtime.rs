use super::*;

impl<'a> Machine<'a> {
    pub(super) fn output_region(&mut self) -> Result<Vec<u8>, Halt> {
        let offset = self.pop()?;
        let size = self.pop()?;
        let (offset, size) = memory_region(offset, size)?;
        self.expand(offset, size)?;
        Ok(self.memory[offset..offset + size].to_vec())
    }

    pub(super) fn expand(&mut self, offset: usize, size: usize) -> Result<(), Halt> {
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

    pub(super) fn charge(&mut self, amount: u64) -> Result<(), Halt> {
        self.gas = self
            .gas
            .checked_sub(amount)
            .ok_or(Halt::Fault("OutOfGas"))?;
        Ok(())
    }

    pub(super) fn pop(&mut self) -> Result<U256, Halt> {
        self.stack.pop().ok_or(Halt::Fault("StackUnderflow"))
    }

    pub(super) fn pop_usize(&mut self) -> Result<usize, Halt> {
        usize_from_word(self.pop()?)
    }

    pub(super) fn pop_u64_saturated(&mut self) -> Result<u64, Halt> {
        let value = self.pop()?;
        Ok(if value > U256::from(u64::MAX) {
            u64::MAX
        } else {
            value.to::<u64>()
        })
    }

    pub(super) fn push(&mut self, value: U256) -> Result<(), Halt> {
        if self.stack.len() >= STACK_LIMIT {
            return Err(Halt::Fault("StackOverflow"));
        }
        self.stack.push(value);
        Ok(())
    }

    pub(super) fn stack_snapshot(&self) -> Vec<String> {
        self.stack
            .iter()
            .map(|value| format!("0x{value:064x}"))
            .collect()
    }
}
