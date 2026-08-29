const test = require("node:test");
const assert = require("node:assert/strict");

test("popup actively injects and opens the Behavior Lens on an Etherscan contract", async () => {
  globalThis.EchoEVMExtension = require("../lib.js");
  const elements = new Map();
  for (const id of ["open-etherscan", "activation-status", "activation-title", "activation-detail"]) {
    elements.set(id, {
      className: "",
      disabled: false,
      textContent: "",
      listeners: {},
      addEventListener(type, listener) { this.listeners[type] = listener; },
      setAttribute(name, value) { this[name] = value; }
    });
  }
  globalThis.document = { getElementById: (id) => elements.get(id) };
  globalThis.window = { close() {} };

  let mounted = false;
  let opened = false;
  const insertedCss = [];
  const injectedScripts = [];
  globalThis.chrome = {
    tabs: {
      query: async () => [{
        id: 17,
        url: "https://etherscan.io/address/0xb07aaBc136EaB64994d3f226c88dd907dF3bf291#code"
      }],
      create: async () => {}
    },
    scripting: {
      insertCSS: async ({ files }) => insertedCss.push(...files),
      executeScript: async ({ files, func }) => {
        if (files) {
          injectedScripts.push(...files);
          mounted = true;
          return [];
        }
        if (String(func).includes(".ee-panel")) {
          opened = true;
          return [{ result: undefined }];
        }
        return [{ result: mounted }];
      }
    }
  };

  require("../popup.js");
  for (let attempt = 0; attempt < 20 && !opened; attempt += 1) {
    await new Promise((resolve) => setImmediate(resolve));
  }

  assert.deepEqual(insertedCss, ["content.css"]);
  assert.deepEqual(injectedScripts, ["lib.js", "contract.js"]);
  assert.equal(opened, true);
  assert.equal(elements.get("activation-title").textContent, "Behavior Lens activated");
  assert.equal(elements.get("open-etherscan").textContent, "Behavior Lens opened");
  assert.equal(elements.get("activation-status")["aria-busy"], "false");
});
