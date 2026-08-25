use alloy_dyn_abi::{FunctionExt, JsonAbiExt, Specifier};
use alloy_json_abi::Function;
use echoevm_core::{
    ExecuteRequest, Fork, build_evidence, decode_hex, execute, infer_behavior, replay_witness,
    trace,
};
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
    #[serde(default = "default_fork")]
    fork: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ContractRequest {
    bytecode: String,
    function: Function,
    #[serde(default)]
    args: Vec<String>,
    #[serde(default = "default_gas")]
    gas_limit: u64,
    #[serde(default = "default_fork")]
    fork: String,
    #[serde(default = "default_profile")]
    profile: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct BehaviorRequest {
    bytecode: String,
    #[serde(default)]
    abi: Vec<Function>,
}

const fn default_gas() -> u64 {
    echoevm_core::DEFAULT_GAS_LIMIT
}

fn default_fork() -> String {
    "Osaka".into()
}

fn default_profile() -> String {
    "auto".into()
}

#[wasm_bindgen(js_name = executeBytecode)]
pub fn execute_json(input: &str) -> Result<String, JsValue> {
    let request: BrowserRequest = serde_json::from_str(input).map_err(js_error)?;
    let result = execute(ExecuteRequest {
        bytecode: decode_hex(&request.bytecode).map_err(js_error)?,
        calldata: decode_hex(&request.calldata).map_err(js_error)?,
        gas_limit: request.gas_limit,
        fork: parse_fork(&request.fork).map_err(js_error)?,
    })
    .map_err(js_error)?;
    serde_json::to_string(&result).map_err(js_error)
}

#[wasm_bindgen(js_name = executeContract)]
pub fn execute_contract_json(input: &str) -> Result<String, JsValue> {
    execute_contract(input).map_err(js_error)
}

#[wasm_bindgen(js_name = inferBehavior)]
pub fn infer_behavior_json(input: &str) -> Result<String, JsValue> {
    infer_browser_behavior(input).map_err(js_error)
}

fn infer_browser_behavior(input: &str) -> Result<String, String> {
    let request: BehaviorRequest =
        serde_json::from_str(input).map_err(|error| error.to_string())?;
    let mut document =
        infer_behavior(&decode_hex(&request.bytecode).map_err(|error| error.to_string())?);
    let signatures = request
        .abi
        .iter()
        .map(|function| {
            (
                format!("0x{}", hex::encode(function.selector())),
                function.signature(),
            )
        })
        .collect::<std::collections::BTreeMap<_, _>>();
    for function in &mut document.functions {
        function.signature = signatures.get(&function.selector).cloned();
    }
    serde_json::to_string(&serde_json::json!({"ok": true, "result": document}))
        .map_err(|error| error.to_string())
}

fn execute_contract(input: &str) -> Result<String, String> {
    let request: ContractRequest =
        serde_json::from_str(input).map_err(|error| error.to_string())?;
    if request.function.state_mutability.as_json_str() != "pure" {
        return Err("local contract execution is limited to ABI functions marked pure".into());
    }
    if request.function.inputs.len() != request.args.len() {
        return Err(format!(
            "argument count mismatch: expected {}, got {}",
            request.function.inputs.len(),
            request.args.len()
        ));
    }
    let values = request
        .function
        .inputs
        .iter()
        .zip(&request.args)
        .map(|(param, value)| {
            param
                .resolve()
                .map_err(|error| error.to_string())?
                .coerce_str(value.trim())
                .map_err(|error| error.to_string())
        })
        .collect::<Result<Vec<_>, _>>()?;
    let calldata = request
        .function
        .abi_encode_input(&values)
        .map_err(|error| error.to_string())?;
    let result = trace(ExecuteRequest {
        bytecode: decode_hex(&request.bytecode).map_err(|error| error.to_string())?,
        calldata: calldata.clone(),
        gas_limit: request.gas_limit,
        fork: parse_fork(&request.fork)?,
    })
    .map_err(|error| error.to_string())?;
    let decoded_output = if result.status == echoevm_protocol::ExecutionStatus::Success {
        let bytes = decode_hex(&result.return_data).map_err(|error| error.to_string())?;
        request
            .function
            .abi_decode_output(&bytes)
            .ok()
            .map(|values| {
                values
                    .into_iter()
                    .map(|value| format!("{value:?}"))
                    .collect::<Vec<_>>()
            })
    } else {
        None
    };
    let total_steps = result.trace.as_ref().map_or(0, Vec::len);
    let evidence = build_evidence(&result, &request.profile, 40);
    serde_json::to_string(&serde_json::json!({
        "ok": true,
        "result": {
            "function": request.function.signature(),
            "calldata": format!("0x{}", hex::encode(calldata)),
            "execution": {
                "engine": result.engine,
                "engineVersion": result.engine_version,
                "status": result.status,
                "returnData": result.return_data,
                "decodedOutput": decoded_output,
                "gasUsed": result.gas_used,
                "totalSteps": total_steps,
                "stateEntries": result.storage.len(),
                "error": result.error,
            },
            "evidence": evidence,
            "warnings": [
                "Local sandbox execution uses empty storage, zero value, and no external contract state.",
                "This result is not a Mainnet transaction simulation, security audit, or formal proof."
            ]
        }
    }))
    .map_err(|error| error.to_string())
}

fn parse_fork(value: &str) -> Result<Fork, String> {
    match value.to_ascii_lowercase().as_str() {
        "cancun" => Ok(Fork::Cancun),
        "prague" => Ok(Fork::Prague),
        "osaka" => Ok(Fork::Osaka),
        _ => Err(format!(
            "unsupported fork {value:?}; expected Cancun, Prague, or Osaka"
        )),
    }
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn contract_execution_encodes_and_runs_a_pure_function() {
        let input = serde_json::json!({
            "bytecode": "6004356024350160005260206000f3",
            "function": {
                "type": "function",
                "name": "add",
                "inputs": [{"name":"a","type":"uint256"},{"name":"b","type":"uint256"}],
                "outputs": [{"name":"","type":"uint256"}],
                "stateMutability": "pure"
            },
            "args": ["2", "3"],
            "fork": "Osaka"
        });
        let encoded = execute_contract(&input.to_string()).expect("execute contract");
        let result: serde_json::Value = serde_json::from_str(&encoded).expect("decode result");
        assert_eq!(result["ok"], true);
        assert_eq!(result["result"]["execution"]["status"], "success");
        assert_eq!(result["result"]["function"], "add(uint256,uint256)");
        assert!(
            result["result"]["calldata"]
                .as_str()
                .unwrap()
                .starts_with("0x771602f7")
        );
    }

    #[test]
    fn contract_execution_rejects_stateful_functions() {
        let input = serde_json::json!({
            "bytecode": "00",
            "function": {"type":"function","name":"read","inputs":[],"outputs":[],"stateMutability":"view"}
        });
        assert!(
            execute_contract(&input.to_string())
                .unwrap_err()
                .contains("marked pure")
        );
    }

    #[test]
    fn behavior_inference_labels_selector_from_abi() {
        let input = serde_json::json!({
            "bytecode": "60003563771602f714600d57005b00",
            "abi": [{
                "type": "function",
                "name": "add",
                "inputs": [{"name":"a","type":"uint256"},{"name":"b","type":"uint256"}],
                "outputs": [{"name":"","type":"uint256"}],
                "stateMutability": "pure"
            }]
        });
        let encoded = infer_browser_behavior(&input.to_string()).expect("infer behavior");
        let result: serde_json::Value = serde_json::from_str(&encoded).expect("decode result");
        assert_eq!(result["ok"], true);
        assert_eq!(result["result"]["schema"], "echoevm.behavior.v1");
        assert_eq!(
            result["result"]["functions"][0]["signature"],
            "add(uint256,uint256)"
        );
    }
}
