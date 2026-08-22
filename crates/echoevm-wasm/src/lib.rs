use echoevm_core::{ExecuteRequest, decode_hex, execute, replay_witness};
use serde::Deserialize;
use wasm_bindgen::prelude::*;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct BrowserRequest {
    bytecode: String,
    #[serde(default)]
    calldata: String,
    #[serde(default = "default_gas")]
    gas_limit: u64,
}

const fn default_gas() -> u64 {
    echoevm_core::DEFAULT_GAS_LIMIT
}

#[wasm_bindgen(js_name = executeBytecode)]
pub fn execute_json(input: &str) -> Result<String, JsValue> {
    let request: BrowserRequest = serde_json::from_str(input).map_err(js_error)?;
    let result = execute(ExecuteRequest {
        bytecode: decode_hex(&request.bytecode).map_err(js_error)?,
        calldata: decode_hex(&request.calldata).map_err(js_error)?,
        gas_limit: request.gas_limit,
        ..Default::default()
    })
    .map_err(js_error)?;
    serde_json::to_string(&result).map_err(js_error)
}

#[wasm_bindgen(js_name = replay)]
pub fn replay_json(input: &str, _options: &str) -> String {
    let response = (|| {
        let witness = echoevm_protocol::ReplayWitness::decode_strict(input.as_bytes())
            .map_err(|error| error.to_string())?;
        let result = replay_witness(&witness, true).map_err(|error| error.to_string())?;
        let total_steps = result.execution.trace.as_ref().map_or(0, Vec::len);
        let state_entries = result.state.len();
        Ok::<_, String>(serde_json::json!({
            "ok": true,
            "result": {
                "transaction": result.transaction,
                "execution": {
                    "engine": result.execution.engine,
                    "engineVersion": result.execution.engine_version,
                    "status": result.execution.status,
                    "returnData": result.execution.return_data,
                    "gasUsed": result.execution.gas_used,
                    "totalSteps": total_steps,
                    "stateEntries": state_entries,
                    "error": result.execution.error,
                },
                "warnings": result.warnings,
                "witness": result.witness,
                "evidence": result.evidence,
            }
        }))
    })();
    match response {
        Ok(value) => value.to_string(),
        Err(error) => serde_json::json!({"ok": false, "error": error}).to_string(),
    }
}

fn js_error(error: impl std::fmt::Display) -> JsValue {
    JsValue::from_str(&error.to_string())
}
