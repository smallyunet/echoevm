(function () {
  "use strict";

  const helpers = globalThis.EchoEVMExtension;
  const contractAddress = helpers.extractContractAddress(window.location.href);
  if (!contractAddress || document.getElementById("echoevm-extension-root")) return;

  discoverContract().then((contract) => {
    if (contract) mountContractLens(contract);
  });

  async function discoverContract() {
    for (let attempt = 0; attempt < 30; attempt += 1) {
      const abiNode = document.querySelector("#js-copytextarea2");
      const codeRoot = document.querySelector("#dividcode");
      if (abiNode && codeRoot) {
        try {
          const abi = helpers.parseContractAbi(abiNode.textContent || "");
          const bytecode = deployedBytecode(codeRoot);
          if (!bytecode) return null;
          const summary = document.querySelector("#ContentPlaceHolder1_contractCodeDiv");
          const functions = helpers.abiFunctions(abi);
          const verification = verificationLabel(summary);
          const implementation = implementationAddress();
          return {
            address: contractAddress,
            abi,
            bytecode,
            functions,
            verification,
            implementation,
            name: summaryValue(summary, "Contract Name") || "Verified contract",
            compiler: summaryValue(summary, "Compiler Version") || "Unknown compiler",
            sourceFiles: Math.max(1, document.querySelectorAll("[data-csource]").length)
          };
        } catch (_) {
          return null;
        }
      }
      await new Promise((resolve) => setTimeout(resolve, 250));
    }
    return null;
  }

  function deployedBytecode(codeRoot) {
    const sections = Array.from(codeRoot.querySelectorAll(".mb-10"));
    const section = sections.find((item) => item.querySelector("h6")?.textContent.trim() === "Deployed Bytecode");
    return helpers.normalizeBytecode(section?.querySelector("pre")?.textContent || "");
  }

  function summaryValue(summary, label) {
    if (!summary) return "";
    const heading = Array.from(summary.querySelectorAll("h6")).find((item) => item.textContent.trim().startsWith(label));
    if (!heading) return "";
    const parent = heading.parentElement;
    return parent?.querySelector("h4, span.text-dark, div.text-nowrap > span")?.textContent.trim() || "";
  }

  function verificationLabel(summary) {
    const text = summary?.textContent || "";
    if (text.includes("Exact Match")) return "Exact Match";
    if (text.includes("Runtime Match")) return "Runtime Match";
    if (text.includes("Similar Match")) return "Similar Match";
    return "Verified source";
  }

  function implementationAddress() {
    const link = document.querySelector("#divImplementationAddress a[href*='/address/']");
    return helpers.extractContractAddress(link?.href || "");
  }

  function mountContractLens(contract) {
    let runnerReady = false;
    let selectedFunction = contract.functions.find((item) => item.stateMutability === "pure") || null;
    const pureFunctions = contract.functions.filter((item) => item.stateMutability === "pure");

    const root = document.createElement("div");
    root.id = "echoevm-extension-root";
    root.innerHTML = `
      <button class="ee-launcher" type="button" aria-expanded="false" aria-controls="echoevm-panel">
        <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3M8 21H5a2 2 0 0 1-2-2v-3m18 0v3a2 2 0 0 1-2 2h-3"/><path d="m9 9 3-2 3 2v4l-3 2-3-2V9Z"/></svg>
        <span><strong>EchoEVM</strong><small>Contract lens</small></span>
        <span class="ee-launcher-status" aria-hidden="true"></span>
      </button>
      <aside class="ee-panel" id="echoevm-panel" aria-labelledby="echoevm-title" hidden>
        <header class="ee-header">
          <div class="ee-brand"><svg viewBox="0 0 24 24" width="22" height="22" aria-hidden="true"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3M8 21H5a2 2 0 0 1-2-2v-3m18 0v3a2 2 0 0 1-2 2h-3"/><path d="m9 9 3-2 3 2v4l-3 2-3-2V9Z"/></svg><div><span>Verified bytecode</span><h2 id="echoevm-title">Contract Lens</h2></div></div>
          <button class="ee-icon-button ee-close" type="button" aria-label="Close EchoEVM panel"><svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18"/></svg></button>
        </header>
        <div class="ee-body">
          <section class="ee-detected" aria-label="Detected contract"><span class="ee-status-dot"></span><div><small>${escapeText(contract.name)}</small><code>${escapeText(helpers.shortHex(contract.address, 12, 10))}</code></div></section>
          <p class="ee-boundary">EchoEVM reads the verified ABI and deployed bytecode from this Etherscan page. Pure functions run locally with empty storage and no external contract state.</p>
          <div class="ee-engine-status" role="status" aria-live="polite"><span class="ee-spinner" aria-hidden="true"></span><span class="ee-engine-copy">Loading local engine…</span></div>
          <dl class="ee-contract-facts">
            <div><dt>Verification</dt><dd>${escapeText(contract.verification)}</dd></div>
            <div><dt>Compiler</dt><dd title="${escapeText(contract.compiler)}">${escapeText(contract.compiler)}</dd></div>
            <div><dt>ABI functions</dt><dd>${helpers.formatInteger(contract.functions.length)}</dd></div>
            <div><dt>Pure runnable</dt><dd>${helpers.formatInteger(pureFunctions.length)}</dd></div>
          </dl>
          <section class="ee-contract-run" aria-labelledby="echoevm-function-title">
            <div class="ee-section-heading"><div><h3 id="echoevm-function-title">Local function execution</h3><p>ABI encoding and EVM execution stay inside the extension.</p></div><span class="ee-local-badge">Local only</span></div>
            ${contract.implementation ? `<div class="ee-notice ee-notice-warning"><strong>Proxy detected</strong><span>Implementation ${escapeText(helpers.shortHex(contract.implementation))}. Local execution is disabled because proxy storage context is not present.</span></div>` : ""}
            ${pureFunctions.length === 0 ? `<div class="ee-notice"><strong>No pure function found</strong><span>The contract can still be inspected here, but stateful calls require explicit prestate.</span></div>` : ""}
            <label class="ee-field" for="echoevm-contract-function"><span>Pure function</span><select id="echoevm-contract-function" ${pureFunctions.length === 0 || contract.implementation ? "disabled" : ""}></select></label>
            <div class="ee-arguments"></div>
            <label class="ee-field" for="echoevm-contract-profile"><span>Evidence question</span><select id="echoevm-contract-profile"><option value="auto">What happened?</option><option value="revert">Why did it fail?</option><option value="arithmetic">Which values drove the result?</option><option value="abi">How was data handled?</option><option value="gas">Where was gas spent?</option></select></label>
            <button class="ee-run" type="button" disabled>Run pure function locally</button>
          </section>
          <div class="ee-error" role="alert" hidden></div>
          <section class="ee-results" aria-labelledby="echoevm-result-title" hidden>
            <div class="ee-verdict"><span class="ee-verdict-mark" aria-hidden="true"></span><div><small>Independent local sandbox</small><h3 id="echoevm-result-title"></h3></div></div>
            <dl class="ee-metrics"></dl><div class="ee-result-detail"></div><div class="ee-warnings" hidden></div>
            <section class="ee-evidence" hidden><div class="ee-section-heading"><div><h3>Selected causal evidence</h3><p class="ee-evidence-note"></p></div></div><ol class="ee-events"></ol></section>
          </section>
        </div>
        <footer class="ee-footer"><span>EchoEVM <b class="ee-version"></b> · Wasm</span><a href="https://github.com/smallyunet/echoevm/blob/main/extensions/chrome/README.md" target="_blank" rel="noreferrer">Execution boundary</a></footer>
      </aside>`;
    document.documentElement.appendChild(root);

    const launcher = root.querySelector(".ee-launcher");
    const panel = root.querySelector(".ee-panel");
    const closeButton = root.querySelector(".ee-close");
    const functionSelect = root.querySelector("#echoevm-contract-function");
    const profile = root.querySelector("#echoevm-contract-profile");
    const argumentsRoot = root.querySelector(".ee-arguments");
    const runButton = root.querySelector(".ee-run");
    const engineStatus = root.querySelector(".ee-engine-status");
    const engineCopy = root.querySelector(".ee-engine-copy");
    const errorBox = root.querySelector(".ee-error");
    const results = root.querySelector(".ee-results");
    root.querySelector(".ee-version").textContent = `v${chrome.runtime.getManifest().version}`;

    for (const item of pureFunctions) {
      const option = document.createElement("option");
      option.value = helpers.functionSignature(item);
      option.textContent = helpers.functionSignature(item);
      functionSelect.appendChild(option);
    }
    renderArguments();

    launcher.addEventListener("click", () => setOpen(panel.hidden));
    closeButton.addEventListener("click", () => setOpen(false));
    functionSelect.addEventListener("change", () => {
      selectedFunction = pureFunctions.find((item) => helpers.functionSignature(item) === functionSelect.value) || null;
      renderArguments();
      clearError();
    });
    runButton.addEventListener("click", runContract);
    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && !panel.hidden) setOpen(false);
    });

    chrome.runtime.sendMessage({ type: "echoevm-engine-status" }, (response) => {
      if (chrome.runtime.lastError || !response?.ready) {
        showError(response?.error || chrome.runtime.lastError?.message || "EchoEVM Wasm failed to start.");
        engineCopy.textContent = "Local engine unavailable";
        return;
      }
      runnerReady = true;
      engineStatus.classList.add("is-ready");
      engineCopy.textContent = "Local engine ready";
      updateRunState();
    });

    function setOpen(open) {
      panel.hidden = !open;
      launcher.setAttribute("aria-expanded", String(open));
      if (open) closeButton.focus({ preventScroll: true });
      else launcher.focus({ preventScroll: true });
    }

    function renderArguments() {
      argumentsRoot.replaceChildren();
      for (const [index, input] of (selectedFunction?.inputs || []).entries()) {
        const label = document.createElement("label");
        label.className = "ee-field";
        const id = `echoevm-argument-${index}`;
        label.htmlFor = id;
        const name = document.createElement("span");
        name.textContent = `${input.name || `Argument ${index + 1}`} · ${input.type}`;
        const field = document.createElement("input");
        field.id = id;
        field.className = "ee-argument";
        field.type = "text";
        field.autocomplete = "off";
        field.spellcheck = false;
        field.placeholder = helpers.inputHint(input);
        field.required = true;
        field.addEventListener("input", updateRunState);
        label.append(name, field);
        argumentsRoot.appendChild(label);
      }
      updateRunState();
    }

    function updateRunState() {
      const inputsReady = Array.from(argumentsRoot.querySelectorAll("input")).every((input) => input.value.trim() !== "");
      runButton.disabled = !runnerReady || !selectedFunction || Boolean(contract.implementation) || !inputsReady;
    }

    function runContract() {
      if (runButton.disabled || !selectedFunction) return;
      clearError();
      results.hidden = true;
      setBusy(true);
      const args = Array.from(argumentsRoot.querySelectorAll("input")).map((input) => input.value.trim());
      chrome.runtime.sendMessage({
        type: "echoevm-contract-execute",
        request: { bytecode: contract.bytecode, function: selectedFunction, args, fork: "Osaka", profile: profile.value, gasLimit: 15_000_000 }
      }, (response) => {
        setBusy(false);
        if (chrome.runtime.lastError || !response?.ok || !response.result) {
          showError(response?.error || chrome.runtime.lastError?.message || "Local contract execution failed.");
          return;
        }
        renderResult(response.result);
      });
    }

    function setBusy(busy) {
      runButton.disabled = busy;
      runButton.textContent = busy ? "Executing in browser…" : "Run pure function locally";
      engineStatus.classList.toggle("is-running", busy);
      engineCopy.textContent = busy ? "Executing with EchoEVM Wasm…" : "Local engine ready";
      if (!busy) updateRunState();
    }

    function showError(message) {
      setBusy(false);
      errorBox.textContent = message;
      errorBox.hidden = false;
      setOpen(true);
    }

    function clearError() {
      errorBox.textContent = "";
      errorBox.hidden = true;
    }

    function renderResult(result) {
      const success = result.execution.status === "success";
      const verdict = root.querySelector(".ee-verdict");
      verdict.classList.toggle("is-success", success);
      verdict.classList.toggle("is-failure", !success);
      root.querySelector(".ee-verdict-mark").textContent = success ? "✓" : "!";
      root.querySelector("#echoevm-result-title").textContent = success ? "Pure function executed" : `Execution ${result.execution.status}`;
      root.querySelector(".ee-metrics").replaceChildren(
        metric("Gas used", helpers.formatInteger(result.execution.gasUsed)),
        metric("Trace steps", helpers.formatInteger(result.execution.totalSteps)),
        metric("State entries", helpers.formatInteger(result.execution.stateEntries)),
        metric("Mode", "Empty state")
      );
      const decoded = Array.isArray(result.execution.decodedOutput) ? result.execution.decodedOutput.join(", ") : "—";
      root.querySelector(".ee-result-detail").replaceChildren(
        detailRow("Function", result.function),
        detailRow("Calldata", result.calldata),
        detailRow("Decoded", decoded),
        detailRow("Raw return", result.execution.returnData)
      );
      const warnings = root.querySelector(".ee-warnings");
      warnings.replaceChildren(...(result.warnings || []).map((warning) => {
        const paragraph = document.createElement("p");
        paragraph.textContent = warning;
        return paragraph;
      }));
      warnings.hidden = !(result.warnings || []).length;
      const events = helpers.evidenceEvents(result);
      root.querySelector(".ee-evidence-note").textContent = `${events.length} of ${helpers.formatInteger(result.evidence?.execution?.totalSteps || result.execution.totalSteps)} steps selected`;
      root.querySelector(".ee-events").replaceChildren(...events.slice(0, 12).map(renderEvidenceEvent));
      root.querySelector(".ee-evidence").hidden = events.length === 0;
      results.hidden = false;
      results.scrollIntoView({ behavior: matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth", block: "nearest" });
    }

    function metric(label, value) {
      const wrapper = document.createElement("div");
      const term = document.createElement("dt");
      const description = document.createElement("dd");
      term.textContent = label;
      description.textContent = value;
      wrapper.append(term, description);
      return wrapper;
    }

    function detailRow(label, value) {
      const row = document.createElement("div");
      const name = document.createElement("span");
      const data = document.createElement("code");
      name.textContent = label;
      data.textContent = value || "—";
      row.append(name, data);
      return row;
    }

    function renderEvidenceEvent(event) {
      const item = document.createElement("li");
      const heading = document.createElement("div");
      const opcode = document.createElement("strong");
      const location = document.createElement("code");
      const explanation = document.createElement("p");
      opcode.textContent = event.op || "OPCODE";
      location.textContent = `step ${event.step} · depth ${event.depth} · pc ${event.pc}`;
      explanation.textContent = event.why || event.error || "Selected execution event";
      heading.append(opcode, location);
      item.append(heading, explanation);
      return item;
    }
  }

  function escapeText(value) {
    const node = document.createElement("span");
    node.textContent = String(value || "");
    return node.innerHTML;
  }
})();
