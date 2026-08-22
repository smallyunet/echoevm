(function () {
  "use strict";

  const helpers = globalThis.EchoEVMExtension;
  const transactionHash = helpers.extractTransactionHash(window.location.href);
  if (!transactionHash || document.getElementById("echoevm-extension-root")) return;

  let runnerReady = false;
  let witnessText = "";
  let activeRequest = "";
  let replayPort = null;

  const root = document.createElement("div");
  root.id = "echoevm-extension-root";
  root.innerHTML = `
    <button class="ee-launcher" type="button" aria-expanded="false" aria-controls="echoevm-panel">
      <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="M12 2 5 6v5c0 5.2 2.8 9.1 7 11 4.2-1.9 7-5.8 7-11V6l-7-4Z"/><path d="m9 12 2 2 4-5"/></svg>
      <span><strong>EchoEVM</strong><small>Local replay</small></span>
      <span class="ee-launcher-status" aria-hidden="true"></span>
    </button>
    <aside class="ee-panel" id="echoevm-panel" aria-labelledby="echoevm-title" hidden>
      <header class="ee-header">
        <div class="ee-brand">
          <svg viewBox="0 0 24 24" width="22" height="22" aria-hidden="true"><path d="M12 2 5 6v5c0 5.2 2.8 9.1 7 11 4.2-1.9 7-5.8 7-11V6l-7-4Z"/><path d="m9 12 2 2 4-5"/></svg>
          <div><span>Browser execution</span><h2 id="echoevm-title">EchoEVM Replay</h2></div>
        </div>
        <button class="ee-icon-button ee-close" type="button" aria-label="Close EchoEVM panel">
          <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18"/></svg>
        </button>
      </header>
      <div class="ee-body">
        <section class="ee-detected" aria-label="Detected transaction">
          <span class="ee-status-dot"></span>
          <div><small>Ethereum transaction detected</small><code class="ee-transaction"></code></div>
        </section>
        <p class="ee-boundary">Execution happens inside this extension with EchoEVM Wasm. A self-contained witness supplies the historical prestate; no CLI or Geth executor is used.</p>
        <div class="ee-engine-status" role="status" aria-live="polite">
          <span class="ee-spinner" aria-hidden="true"></span><span class="ee-engine-copy">Loading local engine…</span>
        </div>
        <section class="ee-import" aria-labelledby="echoevm-import-title">
          <div class="ee-section-heading"><div><h3 id="echoevm-import-title">Replay witness</h3><p>Choose an <code>echoevm.replay-witness.v1</code> JSON file.</p></div><span class="ee-local-badge">Local only</span></div>
          <input class="ee-file-input" id="echoevm-witness-input" type="file" accept="application/json,.json">
          <label class="ee-file-button" for="echoevm-witness-input">
            <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="M12 16V4m0 0L7 9m5-5 5 5M5 14v5h14v-5"/></svg>
            <span><strong>Select witness</strong><small>JSON · maximum 64 MiB</small></span>
          </label>
          <label class="ee-field" for="echoevm-profile"><span>Evidence question</span>
            <select id="echoevm-profile">
              <option value="auto">What happened?</option>
              <option value="revert">Why did it fail?</option>
              <option value="storage">What state changed?</option>
              <option value="call">Which calls mattered?</option>
              <option value="gas">Where was gas spent?</option>
              <option value="abi">What data was returned?</option>
              <option value="arithmetic">Which values drove the result?</option>
            </select>
          </label>
          <button class="ee-run" type="button" disabled>Run local replay</button>
        </section>
        <div class="ee-error" role="alert" hidden></div>
        <section class="ee-results" aria-labelledby="echoevm-result-title" hidden>
          <div class="ee-verdict"><span class="ee-verdict-mark" aria-hidden="true"></span><div><small>Independent EchoEVM execution</small><h3 id="echoevm-result-title"></h3></div></div>
          <dl class="ee-metrics"></dl>
          <div class="ee-result-detail"></div>
          <div class="ee-warnings" hidden></div>
          <section class="ee-evidence" hidden><div class="ee-section-heading"><div><h3>Selected causal evidence</h3><p class="ee-evidence-note"></p></div></div><ol class="ee-events"></ol></section>
        </section>
      </div>
      <footer class="ee-footer"><span>EchoEVM <b class="ee-version"></b> · Wasm</span><a href="https://github.com/smallyunet/echoevm/blob/main/docs/REPLAY_WITNESS.md" target="_blank" rel="noreferrer">Witness format</a></footer>
    </aside>
  `;
  document.documentElement.appendChild(root);

  const launcher = root.querySelector(".ee-launcher");
  const panel = root.querySelector(".ee-panel");
  const closeButton = root.querySelector(".ee-close");
  const fileInput = root.querySelector("#echoevm-witness-input");
  const fileButton = root.querySelector(".ee-file-button");
  const runButton = root.querySelector(".ee-run");
  const profile = root.querySelector("#echoevm-profile");
  const engineStatus = root.querySelector(".ee-engine-status");
  const engineCopy = root.querySelector(".ee-engine-copy");
  const errorBox = root.querySelector(".ee-error");
  const results = root.querySelector(".ee-results");
  root.querySelector(".ee-transaction").textContent = helpers.shortHex(transactionHash, 12, 10);
  root.querySelector(".ee-version").textContent = `v${chrome.runtime.getManifest().version}`;

  launcher.addEventListener("click", () => setOpen(panel.hidden));
  closeButton.addEventListener("click", () => setOpen(false));
  fileInput.addEventListener("change", async () => {
    const file = fileInput.files?.[0];
    if (!file) return;
    await loadWitness(file);
  });
  runButton.addEventListener("click", runReplay);
  fileButton.addEventListener("dragover", (event) => {
    event.preventDefault();
    fileButton.classList.add("is-dragging");
  });
  fileButton.addEventListener("dragleave", () => fileButton.classList.remove("is-dragging"));
  fileButton.addEventListener("drop", async (event) => {
    event.preventDefault();
    fileButton.classList.remove("is-dragging");
    const file = event.dataTransfer?.files?.[0];
    if (file) await loadWitness(file);
  });
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
    runButton.disabled = !witnessText;
  });

  function handleReplayResponse(message) {
    if (message?.type !== "result" || message.id !== activeRequest) return;
    activeRequest = "";
    setBusy(false);
    replayPort?.disconnect();
    replayPort = null;
    if (!message.ok || !message.result) {
      showError(message.error || "EchoEVM replay failed.");
      return;
    }
    if (message.result.transaction?.hash?.toLowerCase() !== transactionHash) {
      showError(`This witness belongs to ${helpers.shortHex(message.result.transaction?.hash)}, not the transaction open on Etherscan.`);
      return;
    }
    renderResult(message.result);
  }

  function setOpen(open) {
    panel.hidden = !open;
    launcher.setAttribute("aria-expanded", String(open));
    if (open) closeButton.focus({ preventScroll: true });
    else launcher.focus({ preventScroll: true });
  }

  async function loadWitness(file) {
    clearError();
    results.hidden = true;
    setOpen(true);
    if (file.size > 64 * 1024 * 1024) {
      showError("The witness exceeds EchoEVM's 64 MiB limit.");
      return;
    }
    try {
      const text = await file.text();
      helpers.validateWitnessText(text);
      witnessText = text;
      fileButton.classList.add("has-file");
      fileButton.querySelector("strong").textContent = file.name;
      fileButton.querySelector("small").textContent = `${helpers.formatInteger(file.size)} bytes · ready to replay`;
      runButton.disabled = !runnerReady;
      if (runnerReady) runReplay();
    } catch (error) {
      witnessText = "";
      runButton.disabled = true;
      showError(error instanceof Error ? error.message : String(error));
    }
  }

  function runReplay() {
    if (!runnerReady || !witnessText) return;
    clearError();
    results.hidden = true;
    activeRequest = crypto.randomUUID();
    setBusy(true);
    replayPort?.disconnect();
    replayPort = chrome.runtime.connect({ name: "echoevm-replay" });
    replayPort.onMessage.addListener(handleReplayResponse);
    replayPort.onDisconnect.addListener(() => {
      if (chrome.runtime.lastError && activeRequest) showError(chrome.runtime.lastError.message);
    });
    replayPort.postMessage({ type: "start", id: activeRequest, options: { profile: profile.value, limit: 40, maxMemoryBytes: 256 } });
    const chunkSize = 256 * 1024;
    for (let offset = 0; offset < witnessText.length; offset += chunkSize) {
      replayPort.postMessage({ type: "chunk", id: activeRequest, data: witnessText.slice(offset, offset + chunkSize) });
    }
    replayPort.postMessage({ type: "execute", id: activeRequest });
  }

  function setBusy(busy) {
    runButton.disabled = busy || !runnerReady || !witnessText;
    runButton.textContent = busy ? "Executing in browser…" : "Run local replay";
    engineStatus.classList.toggle("is-running", busy);
    if (busy) engineCopy.textContent = "Executing with EchoEVM Wasm…";
    else if (runnerReady) engineCopy.textContent = "Local engine ready";
  }

  function showError(message) {
    setBusy(false);
    errorBox.textContent = message;
    errorBox.hidden = false;
  }

  function clearError() {
    errorBox.hidden = true;
    errorBox.textContent = "";
  }

  function renderResult(result) {
    clearError();
    const success = result.execution.status === "success";
    const verdict = root.querySelector(".ee-verdict");
    verdict.classList.toggle("is-success", success);
    verdict.classList.toggle("is-failure", !success);
    root.querySelector("#echoevm-result-title").textContent = success ? "Transaction executed successfully" : `Transaction ${result.execution.status}`;
    root.querySelector(".ee-verdict-mark").textContent = success ? "✓" : "!";

    const metrics = root.querySelector(".ee-metrics");
    metrics.replaceChildren(
      metric("Gas used", helpers.formatInteger(result.execution.gasUsed)),
      metric("Trace steps", helpers.formatInteger(result.execution.totalSteps)),
      metric("State entries", helpers.formatInteger(result.execution.stateEntries)),
      metric("Fork", result.transaction.fork || "—")
    );

    const detail = root.querySelector(".ee-result-detail");
    detail.replaceChildren(detailRow("From", result.transaction.from), detailRow("To", result.transaction.to || "Contract creation"), detailRow("Return data", helpers.shortHex(result.execution.returnData, 18, 14)), detailRow("Witness", helpers.shortHex(result.witness.sha256, 16, 12)));

    const warnings = root.querySelector(".ee-warnings");
    const warningItems = Array.isArray(result.warnings) ? result.warnings : [];
    warnings.replaceChildren(...warningItems.map((warning) => {
      const paragraph = document.createElement("p");
      paragraph.textContent = warning;
      return paragraph;
    }));
    warnings.hidden = warningItems.length === 0;

    const events = helpers.evidenceEvents(result);
    const evidence = root.querySelector(".ee-evidence");
    root.querySelector(".ee-evidence-note").textContent = `${events.length} of ${helpers.formatInteger(result.evidence?.execution?.totalSteps || result.execution.totalSteps)} steps selected`;
    const eventList = root.querySelector(".ee-events");
    eventList.replaceChildren(...events.slice(0, 12).map(renderEvidenceEvent));
    evidence.hidden = events.length === 0;
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
    opcode.textContent = event.opcodeName || "OPCODE";
    location.textContent = `step ${event.step} · depth ${event.depth} · pc ${event.pc}`;
    explanation.textContent = event.explanation || event.error || "Selected execution event";
    heading.append(opcode, location);
    item.append(heading, explanation);
    return item;
  }
})();
