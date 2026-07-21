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
const preset = el('preset');
presets.forEach((item, index) => {
  const option = document.createElement('option'); option.value = index; option.textContent = item[0]; preset.append(option);
});

function applyPreset() {
  const item = presets[Number(preset.value)]; el('bytecode').value = item[1]; el('calldata').value = item[2];
}
preset.addEventListener('change', applyPreset); applyPreset();

el('compare').addEventListener('click', compare);
el('show-equal').addEventListener('change', renderTrace);
el('copy-cli').addEventListener('click', copyCLI);
el('export-json').addEventListener('click', exportJSON);

async function compare() {
  const button = el('compare'); button.disabled = true; button.textContent = 'Comparing…';
  el('request-status').textContent = 'Executing both engines'; el('error').hidden = true;
  try {
    const response = await fetch('/api/diff', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(request())});
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
    lastResult = data; render(data); el('request-status').textContent = 'Comparison complete';
  } catch (error) {
    el('error').textContent = error.message; el('error').hidden = false; el('request-status').textContent = 'Comparison failed';
  } finally {
    button.disabled = false; button.textContent = 'Compare engines';
  }
}

function request() {
  return {fork: 'Cancun', bytecode: el('bytecode').value.trim(), calldata: el('calldata').value.trim(), gasLimit: Number(el('gas').value)};
}

function render(result) {
  el('results').hidden = false;
  el('verdict').textContent = result.match ? 'MATCH' : 'DIVERGENCE';
  el('verdict').className = result.match ? 'match' : 'mismatch';
  el('scope-note').textContent = result.match
    ? 'This input matches in the specified environment. This is not a claim of complete EVM compatibility.'
    : 'The first reliably comparable difference is highlighted below.';
  const items = [
    ['Halt class', result.echoevm.status, result.geth.status, result.statusMatch],
    ['Return data', result.echoevm.returnData, result.geth.returnData, result.returnDataMatch],
    ['Gas used', result.echoevm.gasUsed, result.geth.gasUsed, result.gasMatch],
    ['Storage', `${Object.keys(result.echoevm.storage).length} observed slots`, `${Object.keys(result.geth.storage).length} observed slots`, result.storageMatch],
    ['Trace', `${result.echoevm.trace.length} steps`, `${result.geth.trace.length} steps`, result.traceMatch]
  ];
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
  renderTrace();
}

function renderTrace() {
  if (!lastResult) return;
  const list = el('trace-list'); list.replaceChildren(); const showEqual = el('show-equal').checked;
  const left = lastResult.echoevm.trace, right = lastResult.geth.trace, count = Math.max(left.length, right.length);
  let visible = 0;
  for (let i = 0; i < count; i++) {
    const a = left[i], b = right[i], equal = stepsEqual(a, b);
    if (equal && !showEqual) continue;
    visible++;
    const row = document.createElement('div'); row.className = `trace-row ${equal ? 'equal' : 'different'}`;
    if (lastResult.firstDivergence?.step === i) row.classList.add('first');
    row.append(traceCell(a, i), traceCell(b, i)); list.append(row);
  }
  if (!visible) {
    const empty = document.createElement('p'); empty.className = 'empty'; empty.textContent = `${count} matching steps are folded. Enable “Show matching steps” to inspect them.`; list.append(empty);
  }
}

function traceCell(step, index) {
  const cell = document.createElement('div'); cell.className = 'trace-cell';
  if (!step) { cell.textContent = `Step ${index}: missing`; return cell; }
  const title = document.createElement('div'); title.className = 'trace-title';
  title.append(textNode('span', `#${step.index} · PC ${step.pc}`), textNode('strong', step.opcodeName));
  const details = document.createElement('pre'); details.textContent = `gas ${step.gasBefore} → ${step.gasAfter}\nstack pre  ${formatStack(step.stackBefore)}\nstack post ${formatStack(step.stackAfter)}${step.haltClass ? `\nhalt ${step.haltClass}` : ''}`;
  cell.append(title, details); return cell;
}

function stepsEqual(a, b) {
  if (!a || !b) return false;
  return ['pc','opcode','opcodeName','gasBefore','gasAfter','stackBefore','stackAfter','haltClass'].every(key => JSON.stringify(a[key] ?? null) === JSON.stringify(b[key] ?? null));
}
function formatStack(stack) { return stack ? `[${stack.join(', ')}]` : 'not compared'; }
function formatValue(value) { return typeof value === 'string' ? value : JSON.stringify(value, null, 2); }
function textNode(tag, text) { const node = document.createElement(tag); node.textContent = text; return node; }

async function copyCLI() {
  const r = lastResult.request;
  const command = `echoevm diff --code ${r.bytecode} --input ${r.calldata} --gas ${r.gasLimit} --format text`;
  await navigator.clipboard.writeText(command); el('copy-cli').textContent = 'Copied'; setTimeout(() => el('copy-cli').textContent = 'Copy CLI command', 1200);
}
function exportJSON() {
  const blob = new Blob([JSON.stringify(lastResult, null, 2) + '\n'], {type: 'application/json'}); const url = URL.createObjectURL(blob);
  const link = document.createElement('a'); link.href = url; link.download = 'echoevm-differential.json'; link.click(); URL.revokeObjectURL(url);
}
