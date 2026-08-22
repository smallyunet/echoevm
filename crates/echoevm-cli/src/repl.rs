use anyhow::Result;
use echoevm_core::{ExecuteRequest, assemble, trace};
use std::io::{self, BufRead, Write};

pub fn run() -> Result<()> {
    println!("EchoEVM Rust REPL");
    println!("Enter opcodes (PUSH1 01 ADD) or hex. Type exit to quit.");
    let mut program = Vec::new();
    let stdin = io::stdin();
    let mut lines = stdin.lock().lines();
    loop {
        print!("> ");
        io::stdout().flush()?;
        let Some(line) = lines.next() else { break };
        let line = line?;
        if matches!(line.trim(), "exit" | "quit") {
            break;
        }
        if line.trim().is_empty() {
            continue;
        }
        match assemble(&line) {
            Ok(code) => program.extend(code),
            Err(error) => {
                eprintln!("Error: {error}");
                continue;
            }
        }
        match trace(ExecuteRequest {
            bytecode: program.clone(),
            ..Default::default()
        }) {
            Ok(result) => {
                let stack = result
                    .trace
                    .as_ref()
                    .and_then(|steps| steps.last())
                    .map(|step| &step.stack_before);
                println!(
                    "status={:?} gas={} stack={:?} return={}",
                    result.status, result.gas_used, stack, result.return_data
                );
            }
            Err(error) => eprintln!("Error: {error}"),
        }
    }
    Ok(())
}
