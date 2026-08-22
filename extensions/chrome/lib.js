(function (root) {
  "use strict";

  const TRANSACTION_PATTERN = /^0x[0-9a-fA-F]{64}$/;

  function extractTransactionHash(input) {
    try {
      const url = new URL(input);
      const parts = url.pathname.split("/").filter(Boolean);
      if (parts.length < 2 || parts[0].toLowerCase() !== "tx") return "";
      return TRANSACTION_PATTERN.test(parts[1]) ? parts[1].toLowerCase() : "";
    } catch (_) {
      return "";
    }
  }

  function shortHex(value, head = 8, tail = 6) {
    if (typeof value !== "string" || value.length <= head + tail + 1) return value || "—";
    return `${value.slice(0, head)}…${value.slice(-tail)}`;
  }

  function formatInteger(value) {
    const number = Number(value);
    return Number.isFinite(number) ? new Intl.NumberFormat("en-US").format(number) : "—";
  }

  function validateWitnessText(text, maxBytes = 64 * 1024 * 1024) {
    if (typeof text !== "string" || text.trim() === "") throw new Error("The witness file is empty.");
    if (new TextEncoder().encode(text).byteLength > maxBytes) throw new Error("The witness exceeds EchoEVM's 64 MiB limit.");
    let witness;
    try {
      witness = JSON.parse(text);
    } catch (_) {
      throw new Error("The selected file is not valid JSON.");
    }
    if (witness.schema !== "echoevm.replay-witness.v1") {
      throw new Error("Expected an echoevm.replay-witness.v1 document.");
    }
    return witness;
  }

  function evidenceEvents(result) {
    return Array.isArray(result?.evidence?.events) ? result.evidence.events : [];
  }

  const api = Object.freeze({ extractTransactionHash, shortHex, formatInteger, validateWitnessText, evidenceEvents });
  root.EchoEVMExtension = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(globalThis);
