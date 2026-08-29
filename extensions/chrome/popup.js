"use strict";

const helpers = globalThis.EchoEVMExtension;
const button = document.getElementById("open-etherscan");
const status = document.getElementById("activation-status");
const statusTitle = document.getElementById("activation-title");
const statusDetail = document.getElementById("activation-detail");
let activeTab = null;

initialize().catch((error) => showError(error));

async function initialize() {
  [activeTab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!activeTab?.id || !helpers.extractContractAddress(activeTab.url || "")) {
    setStatus("ready", "Wasm engine included", "Open an Etherscan contract address, then click EchoEVM again.");
    button.disabled = false;
    button.textContent = "Open Etherscan";
    button.addEventListener("click", () => chrome.tabs.create({ url: "https://etherscan.io/contractsVerified" }));
    return;
  }

  button.addEventListener("click", () => activateCurrentContract());
  await activateCurrentContract();
}

async function activateCurrentContract() {
  button.disabled = true;
  button.textContent = "Analyzing current contract…";
  setStatus("running", "Reading deployed bytecode…", "The Behavior Lens will open when local analysis is ready.");

  const target = { tabId: activeTab.id };
  const [{ result: alreadyMounted }] = await chrome.scripting.executeScript({
    target,
    func: () => Boolean(document.getElementById("echoevm-extension-root"))
  });
  if (!alreadyMounted) {
    await chrome.scripting.insertCSS({ target, files: ["content.css"] });
    await chrome.scripting.executeScript({ target, files: ["lib.js", "contract.js"] });
  }

  const mounted = await waitForLens(target);
  if (!mounted) throw new Error("No deployed runtime bytecode was found on this Etherscan page.");
  await chrome.scripting.executeScript({
    target,
    func: () => {
      const panel = document.querySelector("#echoevm-extension-root .ee-panel");
      if (panel?.hidden) document.querySelector("#echoevm-extension-root .ee-launcher")?.click();
    }
  });
  setStatus("ready", "Behavior Lens activated", "Selectors and fallback effects are being inferred locally.");
  button.textContent = "Behavior Lens opened";
  setTimeout(() => window.close(), 450);
}

async function waitForLens(target) {
  for (let attempt = 0; attempt < 120; attempt += 1) {
    const [{ result }] = await chrome.scripting.executeScript({
      target,
      func: () => Boolean(document.getElementById("echoevm-extension-root"))
    });
    if (result) return true;
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  return false;
}

function showError(error) {
  setStatus("error", "Contract analysis did not start", error?.message || String(error));
  button.disabled = false;
  button.textContent = "Retry analysis";
}

function setStatus(state, title, detail) {
  status.className = `status is-${state}`;
  status.setAttribute("aria-busy", state === "running" ? "true" : "false");
  statusTitle.textContent = title;
  statusDetail.textContent = detail;
}
