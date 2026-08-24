use super::*;

impl<'a> Machine<'a> {
    pub(super) fn new(request: Request, state: &'a mut WorldState, address: Address) -> Self {
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

    pub(super) fn run(&mut self) -> Halt {
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
}
