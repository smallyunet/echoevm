const presets = [
  ['ADD return', '60026003015f5260205ff3', '0x'],
  ['Calldata load', '5f355f5260205ff3', '2a00000000000000000000000000000000000000000000000000000000000000'],
  ['Storage roundtrip', '602a5f555f545f5260205ff3', '0x'],
  ['MCOPY', '602a5f5260205f60205e60206020f3', '0x'],
  ['REVERT', '63deadbeef5f526004601cfd', '0x'],
  ['Invalid opcode', 'fe', '0x']
];

const el = id => document.getElementById(id);
let lastResult = null;
let resultMode = 'transaction';
const maxRenderedTraceSteps = 2000;
const preset = el('preset');
presets.forEach((item, index) => {
  const option = document.createElement('option');
  option.value = index; option.textContent = item[0]; preset.append(option);
});

function applyPreset() {
  const item = presets[Number(preset.value)]; el('bytecode').value = item[1]; el('calldata').value = item[2];
}
preset.addEventListener('change', applyPreset); applyPreset();
el('replay').addEventListener('click', replay);
el('transaction-input').addEventListener('keydown', event => { if (event.key === 'Enter') replay(); });
el('compare').addEventListener('click', compare);
el('show-equal').addEventListener('change', renderTrace);
el('copy-cli').addEventListener('click', copyCLI);
el('export-json').addEventListener('click', exportJSON);

async function replay() {
  const input = el('transaction-input').value.trim();
  if (!input) return showError('Enter a transaction hash or Etherscan URL.');
  resultMode = 'transaction';
  await execute({button: el('replay'), loading: 'Loading prestate…', status: 'Fetching transaction, prestate, and Geth trace', endpoint: '/api/replay', body: {input}});
}

async function compare() {
  resultMode = 'bytecode';
  await execute({button: el('compare'), loading: 'Comparing…', status: 'Executing both engines', endpoint: '/api/diff', body: rawRequest()});
}

async function execute({button, loading, status, endpoint, body}) {
  const original = button.textContent; button.disabled = true; button.textContent = loading;
  el('request-status').textContent = status; el('error').hidden = true;
  try {
    const response = await fetch(endpoint, {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)});
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
    lastResult = data; render(data); el('request-status').textContent = resultMode === 'transaction' ? 'Transaction replay complete' : 'Comparison complete';
  } catch (error) {
    showError(error.message); el('request-status').textContent = resultMode === 'transaction' ? 'Transaction replay failed' : 'Comparison failed';
  } finally {
    button.disabled = false; button.textContent = original;
  }
}

function showError(message) { el('error').textContent = message; el('error').hidden = false; }
function rawRequest() { return {fork: 'Cancun', bytecode: el('bytecode').value.trim(), calldata: el('calldata').value.trim(), gasLimit: Number(el('gas').value)}; }

function render(result) {
  el('results').hidden = false;
  el('result-kind').textContent = resultMode === 'transaction' ? 'Transaction replay result' : 'Bytecode comparison result';
  el('verdict').textContent = result.match ? 'MATCH' : 'DIVERGENCE';
  el('verdict').className = result.match ? 'match' : 'mismatch';
  el('scope-note').textContent = result.match
    ? 'EchoEVM matched the Geth reference for this execution.'
    : 'The first reliably comparable difference is highlighted below.';
  renderTransaction(result);
  const items = [
    ['Halt class', result.echoevm.status, result.geth.status, result.statusMatch],
    ['Return data', result.echoevm.returnData, result.geth.returnData, result.returnDataMatch],
    ['Gas used', result.echoevm.gasUsed, result.geth.gasUsed, result.gasMatch],
    ['Trace', `${result.echoevm.trace.length} steps`, `${result.geth.trace.length} steps`, result.traceMatch]
  ];
  if (resultMode === 'bytecode') items.splice(3, 0, ['Storage', `${Object.keys(result.echoevm.storage).length} observed slots`, `${Object.keys(result.geth.storage).length} observed slots`, result.storageMatch]);
  else items.splice(3, 0, ['Post-state', `${Object.keys(result.echoState).length} compared fields`, `${Object.keys(result.gethState).length} compared fields`, result.stateMatch]);
  const summary = el('summary'); summary.replaceChildren();
  items.forEach(item => {
    const card = document.createElement('article'); card.className = `metric ${item[3] ? 'ok' : 'bad'}`;
    card.append(textNode('span', item[0]), textNode('strong', item[3] ? 'MATCH' : 'DIFF'), textNode('code', `Echo ${item[1]}`), textNode('code', `Geth ${item[2]}`)); summary.append(card);
  });
  const d = result.firstDivergence; el('divergence').hidden = !d;
  if (d) {
    const where = d.step === undefined ? 'final result' : `step ${d.step}${d.pc === undefined ? '' : ` · PC ${d.pc}`}${d.opcode ? ` · ${d.opcode}` : ''}`;
    el('divergence-title').textContent = `${where} · ${d.field}`;
    el('div-echo').textContent = `EchoEVM\n${formatValue(d.echoevm)}`; el('div-geth').textContent = `Geth\n${formatValue(d.geth)}`;
  }
  el('trace-note').textContent = result.traceSemantics;
  renderTrace();
  el('results').scrollIntoView({behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'start'});
}

function renderTransaction(result) {
  const summary = el('transaction-summary'); const warnings = el('warnings');
  summary.hidden = resultMode !== 'transaction'; warnings.hidden = true; summary.replaceChildren(); warnings.replaceChildren();
  if (resultMode !== 'transaction') return;
  const tx = result.transaction;
  const heading = document.createElement('div'); heading.className = 'tx-heading';
  const title = document.createElement('div'); title.append(textNode('span', 'Transaction'), externalLink(shortHash(tx.hash), tx.explorerUrl));
  heading.append(title, badge(tx.status), badge(tx.fork)); summary.append(heading);
  const grid = document.createElement('dl'); grid.className = 'tx-grid';
  [['Block', tx.blockNumber], ['From', shortHash(tx.from)], ['To', tx.to ? shortHash(tx.to) : 'Contract creation'], ['Value (wei)', tx.value], ['Gas', `${tx.gasUsed} / ${tx.gasLimit}`], ['Type', tx.type]].forEach(([key, value]) => grid.append(textNode('dt', key), textNode('dd', String(value))));
  summary.append(grid);
  if (result.warnings?.length) {
    result.warnings.forEach(message => { const item = document.createElement('p'); item.textContent = message; warnings.append(item); }); warnings.hidden = false;
  }
}

function renderTrace() {
  if (!lastResult) return;
  const list = el('trace-list'); list.replaceChildren(); const showEqual = el('show-equal').checked;
  const left = lastResult.echoevm.trace, right = lastResult.geth.trace, count = Math.max(left.length, right.length);
  let visible = 0; let omitted = 0;
  for (let i = 0; i < count; i++) {
    const a = left[i], b = right[i], equal = stepsEqual(a, b);
    if (equal && !showEqual) continue;
    if (visible >= maxRenderedTraceSteps) { omitted++; continue; }
    visible++;
    const row = document.createElement('div'); row.className = `trace-row ${equal ? 'equal' : 'different'}`;
    if (lastResult.firstDivergence?.step === i) row.classList.add('first');
    row.append(traceCell(a, i), traceCell(b, i)); list.append(row);
  }
  if (!visible) { const empty = document.createElement('p'); empty.className = 'empty'; empty.textContent = `${count} matching steps are folded. Enable “Show matching steps” to inspect them.`; list.append(empty); }
  else if (omitted) { const note = document.createElement('p'); note.className = 'empty'; note.textContent = `${omitted} additional differing or expanded steps are omitted from the DOM. Export JSON for the complete trace.`; list.append(note); }
}

function traceCell(step, index) {
  const cell = document.createElement('div'); cell.className = 'trace-cell';
  if (!step) { cell.textContent = `Step ${index}: missing`; return cell; }
  const title = document.createElement('div'); title.className = 'trace-title';
  title.append(textNode('span', `#${step.index} · D${step.depth} · PC ${step.pc}`), textNode('strong', step.opcodeName));
  const details = document.createElement('pre'); details.textContent = `gas ${step.gasBefore} → ${step.gasAfter}\nstack pre  ${formatStack(step.stackBefore)}\nstack post ${formatStack(step.stackAfter)}${step.address ? `\naddress ${step.address}` : ''}${step.haltClass ? `\nhalt ${step.haltClass}` : ''}`;
  cell.append(title, details); return cell;
}

function stepsEqual(a, b) {
  if (!a || !b) return false;
  const keys = resultMode === 'transaction' ? ['depth','pc','opcode'] : ['pc','opcode','opcodeName','gasBefore','gasAfter','stackBefore','stackAfter','haltClass'];
  return keys.every(key => JSON.stringify(a[key] ?? null) === JSON.stringify(b[key] ?? null));
}
function formatStack(stack) { return stack ? `[${stack.join(', ')}]` : 'not compared'; }
function formatValue(value) { return typeof value === 'string' ? value : JSON.stringify(value, null, 2); }
function textNode(tag, text) { const node = document.createElement(tag); node.textContent = text; return node; }
function shortHash(value) { return value.length > 18 ? `${value.slice(0, 10)}…${value.slice(-6)}` : value; }
function externalLink(text, href) { const link = document.createElement('a'); link.textContent = text; link.href = href; link.target = '_blank'; link.rel = 'noreferrer'; return link; }
function badge(text) { const node = textNode('span', text); node.className = 'badge'; return node; }

async function copyCLI() {
  let command;
  if (resultMode === 'transaction') command = `echoevm replay ${lastResult.transaction.hash}`;
  else { const r = lastResult.request; command = `echoevm diff --code ${r.bytecode} --input ${r.calldata} --gas ${r.gasLimit} --format text`; }
  await navigator.clipboard.writeText(command); el('copy-cli').textContent = 'Copied'; setTimeout(() => el('copy-cli').textContent = 'Copy CLI command', 1200);
}
function exportJSON() {
  const blob = new Blob([JSON.stringify(lastResult, null, 2) + '\n'], {type: 'application/json'}); const url = URL.createObjectURL(blob);
  const link = document.createElement('a'); link.href = url; link.download = resultMode === 'transaction' ? 'echoevm-replay.json' : 'echoevm-differential.json'; link.click(); URL.revokeObjectURL(url);
}
