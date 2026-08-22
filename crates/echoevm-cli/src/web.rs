use anyhow::{Context, Result};
use echoevm_core::{ExecuteRequest, decode_hex, trace};
use tiny_http::{Header, Method, Response, Server, StatusCode};

const PAGE: &str = r#"<!doctype html><meta charset=utf-8><title>EchoEVM Rust debugger</title>
<style>body{font:15px system-ui;max-width:960px;margin:40px auto;padding:0 20px;background:#0d1525;color:#dce7ff}textarea,pre{width:100%;box-sizing:border-box;background:#121f36;color:#dce7ff;border:1px solid #314466;padding:12px}button{padding:10px 16px;margin:10px 0}</style>
<h1>EchoEVM Rust debugger</h1><textarea id=c rows=5></textarea><button id=r>Run locally</button><pre id=o></pre>
<script>r.onclick=async()=>{o.textContent='running…';o.textContent=JSON.stringify(await (await fetch('/api/run',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({bytecode:c.value})})).json(),null,2)}</script>"#;

pub fn run(addr: &str, initial_code: Option<&str>) -> Result<()> {
    let server = Server::http(addr).map_err(|error| anyhow::anyhow!(error.to_string()))?;
    println!("EchoEVM debugger listening on http://{addr}");
    for mut request in server.incoming_requests() {
        let response = if request.method() == &Method::Post && request.url() == "/api/run" {
            let mut body = String::new();
            request.as_reader().read_to_string(&mut body)?;
            let value: serde_json::Value =
                serde_json::from_str(&body).context("decode web request")?;
            let code = value
                .get("bytecode")
                .and_then(|v| v.as_str())
                .or(initial_code)
                .unwrap_or("");
            match decode_hex(code).and_then(|bytecode| {
                trace(ExecuteRequest {
                    bytecode,
                    ..Default::default()
                })
            }) {
                Ok(result) => json_response(serde_json::to_vec(&result)?, StatusCode(200)),
                Err(error) => json_response(
                    serde_json::to_vec(&serde_json::json!({"error": error.to_string()}))?,
                    StatusCode(400),
                ),
            }
        } else {
            Response::from_string(PAGE).with_header(
                Header::from_bytes("content-type", "text/html; charset=utf-8").unwrap(),
            )
        };
        request.respond(response)?;
    }
    Ok(())
}

fn json_response(bytes: Vec<u8>, status: StatusCode) -> Response<std::io::Cursor<Vec<u8>>> {
    Response::from_data(bytes)
        .with_status_code(status)
        .with_header(Header::from_bytes("content-type", "application/json").unwrap())
}
