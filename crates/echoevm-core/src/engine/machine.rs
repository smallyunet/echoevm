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
                let (storage, control) = self.trace_semantics(op);
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
                    storage,
                    control,
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

    fn trace_semantics(
        &self,
        op: u8,
    ) -> (
        Vec<echoevm_protocol::StorageAccess>,
        Option<echoevm_protocol::ControlFlow>,
    ) {
        use echoevm_protocol::{ControlFlow, StorageAccess};

        let word = |from_top: usize| {
            self.stack
                .len()
                .checked_sub(from_top + 1)
                .and_then(|index| self.stack.get(index))
                .copied()
        };
        let storage = match op {
            0x54 | 0x5c => word(0)
                .map(|slot| {
                    let value = if op == 0x54 {
                        self.state.storage(self.address, slot)
                    } else {
                        self.state
                            .transient
                            .get(&(self.address, slot))
                            .copied()
                            .unwrap_or_default()
                    };
                    vec![StorageAccess {
                        kind: "read".into(),
                        address: self.address.to_string(),
                        slot: format!("0x{slot:064x}"),
                        previous: Some(format!("0x{value:064x}")),
                        value: Some(format!("0x{value:064x}")),
                    }]
                })
                .unwrap_or_default(),
            0x55 | 0x5d => word(0)
                .zip(word(1))
                .map(|(slot, value)| {
                    let previous = if op == 0x55 {
                        self.state.storage(self.address, slot)
                    } else {
                        self.state
                            .transient
                            .get(&(self.address, slot))
                            .copied()
                            .unwrap_or_default()
                    };
                    vec![StorageAccess {
                        kind: "write".into(),
                        address: self.address.to_string(),
                        slot: format!("0x{slot:064x}"),
                        previous: Some(format!("0x{previous:064x}")),
                        value: Some(format!("0x{value:064x}")),
                    }]
                })
                .unwrap_or_default(),
            _ => Vec::new(),
        };
        let control = match op {
            0xf0 | 0xf5 => Some(ControlFlow {
                kind: if op == 0xf0 { "create" } else { "create2" }.into(),
                target: None,
                destination: None,
            }),
            0xf1 | 0xf2 | 0xf4 | 0xfa => Some(ControlFlow {
                kind: match op {
                    0xf1 => "call",
                    0xf2 => "callcode",
                    0xf4 => "delegatecall",
                    _ => "staticcall",
                }
                .into(),
                target: word(1).map(|value| word_address(value).to_string()),
                destination: None,
            }),
            0xf3 => Some(ControlFlow {
                kind: "return".into(),
                target: None,
                destination: None,
            }),
            0xfd => Some(ControlFlow {
                kind: "revert".into(),
                target: None,
                destination: None,
            }),
            _ => None,
        };
        (storage, control)
    }
}
