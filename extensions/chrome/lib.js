(function (root) {
  "use strict";

  const TRANSACTION_PATTERN = /^0x[0-9a-fA-F]{64}$/;
  const ADDRESS_PATTERN = /^0x[0-9a-fA-F]{40}$/;
  const BYTECODE_PATTERN = /^0x[0-9a-fA-F]+$/;

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

  function extractContractAddress(input) {
    try {
      const url = new URL(input);
      const parts = url.pathname.split("/").filter(Boolean);
      if (parts.length < 2 || parts[0].toLowerCase() !== "address") return "";
      return ADDRESS_PATTERN.test(parts[1]) ? parts[1].toLowerCase() : "";
    } catch (_) {
      return "";
    }
  }

  function normalizeBytecode(input) {
    if (typeof input !== "string") return "";
    const compact = input.replace(/\s+/g, "");
    const prefixed = compact.startsWith("0x") ? compact : `0x${compact}`;
    return BYTECODE_PATTERN.test(prefixed) && prefixed.length % 2 === 0 ? prefixed.toLowerCase() : "";
  }

  function parseContractAbi(input) {
    let abi;
    try {
      abi = JSON.parse(input);
    } catch (_) {
      throw new Error("Etherscan's contract ABI is not valid JSON.");
    }
    if (!Array.isArray(abi)) throw new Error("Etherscan's contract ABI is not an array.");
    return abi;
  }

  function abiFunctions(abi) {
    if (!Array.isArray(abi)) return [];
    return abi.filter((item) => item && item.type === "function" && typeof item.name === "string");
  }

  function functionSignature(item) {
    const inputs = Array.isArray(item?.inputs) ? item.inputs.map((input) => input.type || "?").join(",") : "";
    return `${item?.name || "function"}(${inputs})`;
  }

  function inputHint(input) {
    const type = input?.type || "value";
    if (type === "address") return "0x…";
    if (type === "bool") return "true or false";
    if (type === "string") return "text";
    if (type === "bytes") return "0x…";
    if (/^bytes\d+$/.test(type)) return `${type} as 0x…`;
    if (/^(u?int)\d*$/.test(type)) return "integer";
    if (type.endsWith("[]")) return "[value1,value2]";
    return type;
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

  const api = Object.freeze({
    extractTransactionHash,
    extractContractAddress,
    normalizeBytecode,
    parseContractAbi,
    abiFunctions,
    functionSignature,
    inputHint,
    shortHex,
    formatInteger,
    validateWitnessText,
    evidenceEvents
  });
  root.EchoEVMExtension = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(globalThis);
