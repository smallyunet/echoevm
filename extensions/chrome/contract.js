(function () {
  "use strict";

  const helpers = globalThis.EchoEVMExtension;
  const contractAddress = helpers.extractContractAddress(window.location.href);
  if (!contractAddress || document.getElementById("echoevm-extension-root")) return;

  discoverContract().then((contract) => {
    if (contract) mountContractLens(contract);
  }).catch((error) => console.warn("EchoEVM contract discovery failed", error));

  async function discoverContract() {
    return helpers.pollForValue(() => {
      const abiNode = document.querySelector("#js-copytextarea2");
      const codeRoot = document.querySelector("#dividcode, #ContentPlaceHolder1_contractCodeDiv");
      if (!codeRoot) return null;
      const bytecode = deployedBytecode(codeRoot);
      if (!bytecode) return null;
      let abi = [];
      if (abiNode?.textContent?.trim()) {
        try {
          abi = helpers.parseContractAbi(abiNode.textContent);
        } catch (_) {
          abi = [];
        }
      }
      const summary = document.querySelector("#ContentPlaceHolder1_contractCodeDiv");
      const functions = helpers.abiFunctions(abi);
      const verification = verificationLabel(summary);
      const implementation = implementationAddress();
      return {
        address: contractAddress,
        abi,
        bytecode,
        functions,
        verification: abiNode ? verification : "Runtime bytecode",
        implementation,
        name: summaryValue(summary, "Contract Name") || "Deployed contract",
        compiler: summaryValue(summary, "Compiler Version") || "Unknown compiler",
        sourceFiles: Math.max(1, document.querySelectorAll("[data-csource]").length)
      };
    });
  }

  function deployedBytecode(codeRoot) {
    const sections = Array.from(codeRoot.querySelectorAll(".mb-10"));
    const section = sections.find((item) => /^(Deployed|Runtime) Bytecode$/i.test(item.querySelector("h6")?.textContent.trim() || ""));
    if (section) return helpers.normalizeBytecode(section.querySelector("pre")?.textContent || "");
    const heading = Array.from(document.querySelectorAll("h5, h6")).find((item) => /^(Deployed|Runtime) Bytecode$/i.test(item.textContent.trim()));
    return helpers.normalizeBytecode(heading?.parentElement?.parentElement?.querySelector("pre")?.textContent || "");
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
    if (document.getElementById("echoevm-extension-root")) return;
    let runnerReady = false;
    let selectedFunction = contract.functions.find((item) => item.stateMutability === "pure") || null;
    const pureFunctions = contract.functions.filter((item) => item.stateMutability === "pure");

    const root = document.createElement("div");
    root.id = "echoevm-extension-root";
    root.innerHTML = `
      <button class="ee-launcher" type="button" aria-expanded="false" aria-controls="echoevm-panel">
        <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3M8 21H5a2 2 0 0 1-2-2v-3m18 0v3a2 2 0 0 1-2 2h-3"/><path d="m9 9 3-2 3 2v4l-3 2-3-2V9Z"/></svg>
        <span><strong>EchoEVM</strong><small>Behavior lens</small></span>
        <span class="ee-launcher-status" aria-hidden="true"></span>
      </button>
      <aside class="ee-panel" id="echoevm-panel" aria-labelledby="echoevm-title" hidden>
        <header class="ee-header">
          <div class="ee-brand"><svg viewBox="0 0 24 24" width="22" height="22" aria-hidden="true"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3M8 21H5a2 2 0 0 1-2-2v-3m18 0v3a2 2 0 0 1-2 2h-3"/><path d="m9 9 3-2 3 2v4l-3 2-3-2V9Z"/></svg><div><span>Deployed bytecode</span><h2 id="echoevm-title">Behavior Lens</h2></div></div>
          <button class="ee-icon-button ee-close" type="button" aria-label="Close EchoEVM panel"><svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18"/></svg></button>
        </header>
        <div class="ee-body">
          <section class="ee-detected" aria-label="Detected contract"><span class="ee-status-dot"></span><div><small>${escapeText(contract.name)}</small><code>${escapeText(helpers.shortHex(contract.address, 12, 10))}</code></div></section>
          <p class="ee-boundary">EchoEVM reads deployed bytecode directly from this Etherscan page and infers bounded behavioral effects locally. A verified ABI, when present, only supplies function names.</p>
          <div class="ee-engine-status" role="status" aria-live="polite"><span class="ee-spinner" aria-hidden="true"></span><span class="ee-engine-copy">Loading local engine…</span></div>
          <dl class="ee-contract-facts">
            <div><dt>Verification</dt><dd>${escapeText(contract.verification)}</dd></div>
            <div><dt>Compiler</dt><dd title="${escapeText(contract.compiler)}">${escapeText(contract.compiler)}</dd></div>
            <div><dt>ABI functions</dt><dd>${helpers.formatInteger(contract.functions.length)}</dd></div>
            <div><dt>Bytecode</dt><dd>${helpers.formatInteger((contract.bytecode.length - 2) / 2)} bytes</dd></div>
          </dl>
          <section class="ee-behavior" aria-labelledby="echoevm-behavior-title">
            <div class="ee-section-heading"><div><h3 id="echoevm-behavior-title">Behavioral ABI</h3><p>Selectors, reachable effects, value origins, and coverage.</p></div><span class="ee-local-badge">Auto · local</span></div>
            <div class="ee-behavior-status" role="status" aria-live="polite"><span class="ee-spinner" aria-hidden="true"></span><span>Waiting for the local engine…</span></div>
            <div class="ee-behavior-result" hidden>
              <dl class="ee-metrics ee-behavior-metrics"></dl>
              <div class="ee-capabilities" aria-label="Inferred contract capabilities"></div>
              <div class="ee-function-list"></div>
              <div class="ee-behavior-limitations"></div>
            </div>
          </section>
          <section class="ee-contract-run" aria-labelledby="echoevm-function-title">
            <div class="ee-section-heading"><div><h3 id="echoevm-function-title">Local function execution</h3><p>ABI encoding and EVM execution stay inside the extension.</p></div><span class="ee-local-badge">Local only</span></div>
            ${contract.implementation ? `<div class="ee-notice ee-notice-warning"><strong>Proxy detected</strong><span>Implementation ${escapeText(helpers.shortHex(contract.implementation))}. Local execution is disabled because proxy storage context is not present.</span></div>` : ""}
            ${pureFunctions.length === 0 ? `<div class="ee-notice"><strong>No runnable pure function</strong><span>Behavior inference still works from bytecode alone. Stateful execution requires explicit prestate.</span></div>` : ""}
            <label class="ee-field" for="echoevm-contract-function"><span>Pure function</span><select id="echoevm-contract-function" ${pureFunctions.length === 0 || contract.implementation ? "disabled" : ""}></select></label>
            <div class="ee-arguments"></div>
            <label class="ee-field" for="echoevm-contract-profile"><span>Evidence question</span><select id="echoevm-contract-profile"><option value="auto">What happened?</option><option value="revert">Why did it fail?</option><option value="arithmetic">Which values drove the result?</option><option value="abi">How was data handled?</option><option value="gas">Where was gas spent?</option></select></label>
            <button class="ee-run" type="button" disabled>Run pure function locally</button>
          </section>
          <div class="ee-error" role="alert" hidden></div>
          <section class="ee-results" aria-labelledby="echoevm-result-title" hidden>
            <div class="ee-verdict"><span class="ee-verdict-mark" aria-hidden="true"></span><div><small>Independent local sandbox</small><h3 id="echoevm-result-title"></h3></div></div>
            <dl class="ee-metrics"></dl><div class="ee-result-detail"></div><div class="ee-warnings" hidden></div>
            <section class="ee-evidence" hidden><div class="ee-section-heading"><div><h3>Selected execution evidence</h3><p class="ee-evidence-note"></p></div></div><ol class="ee-events"></ol></section>
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
    const behaviorStatus = root.querySelector(".ee-behavior-status");
    const behaviorResult = root.querySelector(".ee-behavior-result");
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
        behaviorStatus.classList.add("is-error");
        behaviorStatus.querySelector("span:last-child").textContent = "Local behavior engine unavailable";
        return;
      }
      runnerReady = true;
      engineStatus.classList.add("is-ready");
      engineCopy.textContent = "Local engine ready";
      updateRunState();
      runBehaviorInference();
    });

    function runBehaviorInference() {
      behaviorStatus.classList.add("is-running");
      behaviorStatus.querySelector("span:last-child").textContent = "Inferring behavioral effects from bytecode…";
      chrome.runtime.sendMessage({
        type: "echoevm-infer-behavior",
        request: { bytecode: contract.bytecode, abi: contract.abi }
      }, (response) => {
        behaviorStatus.classList.remove("is-running");
        if (chrome.runtime.lastError || !response?.ok || !response.result) {
          behaviorStatus.classList.add("is-error");
          behaviorStatus.querySelector("span:last-child").textContent = response?.error || chrome.runtime.lastError?.message || "Behavior inference failed.";
          return;
        }
        renderBehavior(response.result);
      });
    }

    function renderBehavior(behavior) {
      const coverage = behavior.coverage || {};
      behaviorStatus.classList.add("is-ready");
      behaviorStatus.querySelector("span:last-child").textContent = `Behavioral ABI ready · ${behavior.schema}`;
      root.querySelector(".ee-behavior-metrics").replaceChildren(
        metric("Selectors", helpers.formatInteger(coverage.recognizedSelectors)),
        metric("Effects", helpers.formatInteger(behavior.contractEffects?.length || 0)),
        metric("Reachable ops", helpers.formatInteger(coverage.reachableInstructions)),
        metric("Unresolved jumps", helpers.formatInteger(coverage.unresolvedJumps))
      );

      const capabilityRoot = root.querySelector(".ee-capabilities");
      const capabilityItems = behavior.contractCapabilities || [];
      capabilityRoot.replaceChildren(...capabilityItems.map((capability) => {
        const badge = document.createElement("span");
        badge.textContent = capabilityLabel(capability);
        return badge;
      }));
      capabilityRoot.hidden = capabilityItems.length === 0;

      const functions = Array.isArray(behavior.functions) ? behavior.functions : [];
      const functionRoot = root.querySelector(".ee-function-list");
      functionRoot.replaceChildren(...functions.slice(0, 40).map(renderBehaviorFunction));
      if (functions.length > 40) {
        const note = document.createElement("p");
        note.className = "ee-list-note";
        note.textContent = `${helpers.formatInteger(functions.length - 40)} additional selectors are available in the JSON protocol output.`;
        functionRoot.appendChild(note);
      }

      const limitations = root.querySelector(".ee-behavior-limitations");
      limitations.replaceChildren(...(behavior.limitations || []).map((limitation) => {
        const paragraph = document.createElement("p");
        paragraph.textContent = limitation;
        return paragraph;
      }));
      behaviorResult.hidden = false;
    }

    function renderBehaviorFunction(item) {
      const details = document.createElement("details");
      details.className = "ee-behavior-function";
      const summary = document.createElement("summary");
      const identity = document.createElement("span");
      const title = document.createElement("strong");
      const selector = document.createElement("code");
      title.textContent = item.signature || item.selector;
      selector.textContent = item.signature ? item.selector : `entry pc ${item.entryPc}`;
      identity.append(title, selector);
      const count = document.createElement("span");
      count.className = "ee-effect-count";
      count.textContent = `${item.effects?.length || 0} effects`;
      summary.append(identity, count);
      details.appendChild(summary);

      const body = document.createElement("div");
      body.className = "ee-behavior-function-body";
      const capabilities = document.createElement("p");
      capabilities.textContent = (item.capabilities || []).map(capabilityLabel).join(" · ") || "No tracked state or call effect reached";
      body.appendChild(capabilities);
      for (const effect of (item.effects || []).slice(0, 12)) {
        const row = document.createElement("div");
        const heading = document.createElement("span");
        const location = document.createElement("code");
        const inputs = document.createElement("code");
        heading.textContent = effectLabel(effect.kind);
        location.textContent = `${effect.opcode} · pc ${effect.pc}`;
        inputs.textContent = Object.entries(effect.inputs || {}).map(([name, value]) => `${name}=${value}`).join(" · ") || "No recovered value origin";
        row.append(heading, location, inputs);
        body.appendChild(row);
      }
      if ((item.effects || []).length > 12) {
        const note = document.createElement("p");
        note.textContent = `${helpers.formatInteger(item.effects.length - 12)} additional effects omitted from this compact view.`;
        body.appendChild(note);
      }
      details.appendChild(body);
      return details;
    }

    function capabilityLabel(value) {
      return ({
        "reads-persistent-state": "Reads storage",
        "writes-persistent-state": "Writes storage",
        "uses-transient-state": "Uses transient storage",
        "calls-external-code": "Calls external code",
        "executes-delegate-code": "Delegate execution",
        "reads-external-state": "Reads external state",
        "creates-contracts": "Creates contracts",
        "can-self-destruct": "Self-destruct path",
        "emits-events": "Emits events"
      })[value] || value;
    }

    function effectLabel(value) {
      return ({
        "storage-read": "Storage read",
        "storage-write": "Storage write",
        "transient-read": "Transient read",
        "transient-write": "Transient write",
        "external-call": "External call",
        "delegate-call": "Delegate call",
        "static-call": "Static call",
        "contract-create": "Contract creation",
        "contract-create2": "Deterministic creation",
        "event-log": "Event log",
        "self-destruct": "Self destruct",
        "balance-read": "Balance read",
        "external-code-read": "External code read"
      })[value] || value;
    }

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
