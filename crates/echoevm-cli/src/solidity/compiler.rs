use super::*;

pub(super) fn compile(args: &CompilerArgs) -> Result<(Vec<CompiledContract>, String)> {
    let source = fs::canonicalize(&args.source)
        .with_context(|| format!("read Solidity source {}", args.source.display()))?;
    let base = fs::canonicalize(
        args.base_path
            .clone()
            .unwrap_or_else(|| source.parent().unwrap_or(Path::new(".")).to_owned()),
    )?;
    let key = source
        .strip_prefix(&base)
        .unwrap_or(&source)
        .to_string_lossy()
        .replace('\\', "/");
    let content = fs::read_to_string(&source)?;
    let input = json!({
        "language": "Solidity",
        "sources": { key.clone(): { "content": content } },
        "settings": {
            "optimizer": { "enabled": args.optimize, "runs": args.optimizer_runs },
            "viaIR": args.via_ir,
            "remappings": args.remappings,
            "evmVersion": "cancun",
            "outputSelection": { "*": { "*": ["abi", "evm.bytecode.object", "evm.deployedBytecode.object", "evm.deployedBytecode.sourceMap"], "": ["ast"] } }
        }
    });
    let mut command = Command::new(&args.solc);
    command
        .args(&args.solc_args)
        .arg("--standard-json")
        .arg("--base-path")
        .arg(&base);
    for path in &args.include_paths {
        command.arg("--include-path").arg(fs::canonicalize(path)?);
    }
    let mut child = command
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .with_context(|| format!("start Solidity compiler {}", args.solc))?;
    child
        .stdin
        .take()
        .expect("piped stdin")
        .write_all(&serde_json::to_vec(&input)?)?;
    let output = child.wait_with_output()?;
    if !output.status.success() {
        bail!(
            "solc compilation failed: {}",
            String::from_utf8_lossy(&output.stderr).trim()
        );
    }
    let compiled: SolcOutput =
        serde_json::from_slice(&output.stdout).context("parse solc standard JSON output")?;
    let errors: Vec<_> = compiled
        .errors
        .iter()
        .filter(|error| error.severity == "error")
        .map(|error| error.formatted_message.as_deref().unwrap_or(&error.message))
        .collect();
    if !errors.is_empty() {
        bail!("solc compilation failed: {}", errors.join("\n"));
    }
    let source_names: BTreeMap<_, _> = compiled
        .sources
        .iter()
        .map(|(name, source)| (source.id, name.clone()))
        .collect();
    let mut contracts = Vec::new();
    for (source_name, by_name) in compiled.contracts {
        for (name, artifact) in by_name {
            if artifact.evm.deployed_bytecode.object.is_empty() {
                continue;
            }
            let abi: JsonAbi = serde_json::from_value(artifact.abi)?;
            let locations = compiled
                .sources
                .get(&source_name)
                .map(|source| function_locations(&source.ast, &name, &source_names))
                .unwrap_or_default();
            contracts.push(CompiledContract {
                key: format!("{source_name}:{name}"),
                name,
                initcode: artifact.evm.bytecode.object,
                runtime: artifact.evm.deployed_bytecode.object,
                source_map: artifact.evm.deployed_bytecode.source_map,
                abi,
                source_names: source_names.clone(),
                function_locations: locations,
            });
        }
    }
    contracts.sort_by(|a, b| a.key.cmp(&b.key));
    if contracts.is_empty() {
        bail!("solc produced no deployable contracts");
    }
    Ok((contracts, compiler_version(args)))
}

pub(super) fn compiler_version(args: &CompilerArgs) -> String {
    Command::new(&args.solc)
        .args(&args.solc_args)
        .arg("--version")
        .output()
        .ok()
        .map(|output| {
            String::from_utf8_lossy(&output.stdout)
                .lines()
                .last()
                .unwrap_or("unknown")
                .trim()
                .trim_start_matches("Version:")
                .trim()
                .to_owned()
        })
        .filter(|value| !value.is_empty())
        .unwrap_or_else(|| "unknown".into())
}

#[derive(Deserialize)]
struct SolcOutput {
    #[serde(default)]
    contracts: BTreeMap<String, BTreeMap<String, SolcContract>>,
    #[serde(default)]
    sources: BTreeMap<String, SolcSource>,
    #[serde(default)]
    errors: Vec<SolcError>,
}

#[derive(Deserialize)]
struct SolcContract {
    abi: Value,
    evm: SolcEvm,
}
#[derive(Deserialize)]
struct SolcEvm {
    bytecode: SolcBytecode,
    #[serde(rename = "deployedBytecode")]
    deployed_bytecode: SolcDeployedBytecode,
}
#[derive(Deserialize)]
struct SolcBytecode {
    #[serde(default)]
    object: String,
}
#[derive(Deserialize)]
struct SolcDeployedBytecode {
    #[serde(default)]
    object: String,
    #[serde(default, rename = "sourceMap")]
    source_map: String,
}
#[derive(Deserialize)]
struct SolcSource {
    id: i64,
    #[serde(default)]
    ast: Value,
}
#[derive(Deserialize)]
struct SolcError {
    severity: String,
    message: String,
    #[serde(rename = "formattedMessage")]
    formatted_message: Option<String>,
}
