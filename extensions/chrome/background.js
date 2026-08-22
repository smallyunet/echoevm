"use strict";

import initEchoEVM, { replay } from "./wasm/engine.js";

const maxWitnessCharacters = 64 * 1024 * 1024;
const enginePromise = bootEngine();

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type !== "echoevm-engine-status") return false;
  enginePromise.then(
    () => sendResponse({ ready: true }),
    (error) => sendResponse({ ready: false, error: normalizeError(error) })
  );
  return true;
});

chrome.runtime.onConnect.addListener((port) => {
  if (port.name !== "echoevm-replay") return;
  let request = null;
  let chunks = [];
  let characters = 0;

  port.onMessage.addListener((message) => {
    if (message?.type === "start" && typeof message.id === "string") {
      request = { id: message.id, options: message.options || {} };
      chunks = [];
      characters = 0;
      return;
    }
    if (!request || message?.id !== request.id) return;
    if (message.type === "chunk" && typeof message.data === "string") {
      characters += message.data.length;
      if (characters > maxWitnessCharacters) {
        postError(port, request.id, "The witness exceeds the extension transfer limit.");
        request = null;
        chunks = [];
        return;
      }
      chunks.push(message.data);
      return;
    }
    if (message.type === "execute") {
      const current = request;
      const witness = chunks.join("");
      request = null;
      chunks = [];
      characters = 0;
      executeReplay(port, current, witness);
    }
  });

  port.onDisconnect.addListener(() => {
    request = null;
    chunks = [];
    characters = 0;
  });
});

async function bootEngine() {
  await initEchoEVM({ module_or_path: chrome.runtime.getURL("wasm/engine_bg.wasm") });
}

async function executeReplay(port, request, witness) {
  try {
    await enginePromise;
    const encoded = replay(witness, JSON.stringify(request.options));
    const response = JSON.parse(encoded);
    port.postMessage({ type: "result", id: request.id, ...response });
  } catch (error) {
    postError(port, request.id, normalizeError(error));
  }
}

function postError(port, id, error) {
  try {
    port.postMessage({ type: "result", id, ok: false, error });
  } catch (_) {
    // The page may have closed while replay was executing.
  }
}

function normalizeError(error) {
  return error instanceof Error ? error.message : String(error || "Unknown EchoEVM Wasm error.");
}
