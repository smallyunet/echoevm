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
el('copy-link').addEventListener('click', copyShareLink);
el('copy-evidence').addEventListener('click', copyEvidence);
el('copy-cli').addEventListener('click', copyCLI);
el('export-json').addEventListener('click', exportJSON);
loadRecentTransactions();
hydrateDeepLink();

async function loadRecentTransactions() {
  const list = el('recent-transactions'); const status = el('recent-status');
  try {
    const response = await fetch('/api/recent-transactions', {headers: {'Accept': 'application/json'}, cache: 'no-store'});
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
    list.replaceChildren(); list.setAttribute('aria-busy', 'false');
    status.textContent = `Ethereum block ${data.blockNumber}`;
    if (!data.transactions?.length) return renderRecentEmpty('The latest block has no transactions. Paste a hash instead.');
    data.transactions.forEach(transaction => {
      const button = document.createElement('button'); button.type = 'button'; button.className = 'recent-transaction';
      button.setAttribute('aria-pressed', 'false'); button.setAttribute('aria-label', `Use transaction ${transaction.hash}`); button.title = transaction.hash;
      button.append(textNode('code', shortHash(transaction.hash)), textNode('span', `Transaction #${transaction.transactionIndex}`));
      button.addEventListener('click', () => selectRecentTransaction(transaction.hash, button)); list.append(button);
    });
  } catch (_) {
    list.setAttribute('aria-busy', 'false'); status.textContent = 'Recent transactions unavailable';
    renderRecentEmpty('Paste a confirmed transaction hash or Etherscan URL instead.');
  }
}

function renderRecentEmpty(message) {
  const list = el('recent-transactions'); list.replaceChildren();
  const empty = textNode('p', message); empty.className = 'recent-empty'; list.append(empty);
}

function selectRecentTransaction(hash, selected) {
  el('transaction-input').value = hash;
  document.querySelectorAll('.recent-transaction').forEach(button => button.setAttribute('aria-pressed', String(button === selected)));
  el('request-status').textContent = `Selected ${shortHash(hash)}. Press Verify & explain to run it.`;
  el('error').hidden = true;
}

async function replay() {
  const input = el('transaction-input').value.trim();
  if (!input) return showError('Enter a transaction hash or Etherscan URL.');
  resultMode = 'transaction';
  const profile = el('evidence-profile').value;
  await execute({button: el('replay'), loading: 'Verifying…', status: 'Executing with EchoEVM and comparing the optional RPC reference', endpoint: '/api/verify', body: {input, profile, limit: 40, maxMemoryBytes: 256}});
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
    lastResult = data; render(data);
    if (resultMode === 'transaction') updateShareURL(data.transaction.hash, data.evidence?.profile || el('evidence-profile').value);
    el('request-status').textContent = resultMode === 'transaction' ? 'Transaction explanation complete' : 'Comparison complete';
  } catch (error) {
    showError(error.message); el('request-status').textContent = resultMode === 'transaction' ? 'Transaction verification failed' : 'Comparison failed';
  } finally {
    button.disabled = false; button.textContent = original;
  }
}

function showError(message) { el('error').textContent = message; el('error').hidden = false; }
function rawRequest() { return {fork: 'Osaka', bytecode: el('bytecode').value.trim(), calldata: el('calldata').value.trim(), gasLimit: Number(el('gas').value)}; }

function render(result) {
  el('results').hidden = false;
  const evidence = resultMode === 'transaction' ? result.evidence : null;
  el('result-kind').textContent = evidence ? 'Verified causal execution evidence' : resultMode === 'transaction' ? 'Transaction verification result' : 'Bytecode comparison result';
  const status = evidence?.execution?.status;
  el('verdict').textContent = evidence ? `${String(status || 'unknown').toUpperCase()} EXPLAINED` : result.match ? 'MATCH' : 'DIVERGENCE';
  el('verdict').className = evidence ? (status === 'success' ? 'match' : 'mismatch') : result.match ? 'match' : 'mismatch';
  el('scope-note').textContent = evidence
    ? `EchoEVM selected ${evidence.selection.selected} causal events from ${evidence.execution.totalSteps} executed steps. The limit changes presentation, not execution.`
    : result.match ? 'EchoEVM matched the Geth reference for this execution.' : 'The first reliably comparable difference is highlighted below.';
  renderTransaction(result);
  renderEvidence(evidence);
  el('copy-link').hidden = !evidence;
  el('copy-evidence').hidden = !evidence;
  el('comparison-details').open = resultMode === 'bytecode';
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
    el('divergence-title').textContent = `${where} · ${formatDivergenceField(d.field)}`;
    el('div-echo').textContent = `EchoEVM\n${formatValue(d.echoevm)}`; el('div-geth').textContent = `Geth\n${formatValue(d.geth)}`;
  }
  el('trace-note').textContent = result.traceSemantics;
  renderTrace();
  el('results').scrollIntoView({behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'start'});
}

function renderEvidence(evidence) {
  const section = el('evidence'); const events = el('evidence-events'); const links = el('evidence-links');
  section.hidden = !evidence; events.replaceChildren(); links.replaceChildren(); links.hidden = true;
  if (!evidence) return;
  el('evidence-note').textContent = `${profileLabel(evidence.profile)} · ${evidence.schema}`;
  const selection = evidence.selection;
  el('evidence-count').textContent = `${selection.selected} selected · ${selection.omitted} omitted${selection.truncated ? ' · bounded' : ''}`;
  if (!evidence.events?.length) {
    const empty = textNode('p', 'No opcode event matched this profile. The transaction metadata and optional comparison still describe the execution result.');
    empty.className = 'evidence-empty'; events.append(empty);
  } else {
    evidence.events.forEach(event => {
      const card = document.createElement('article'); card.className = 'evidence-event';
      const header = document.createElement('div'); header.className = 'evidence-event-head';
      header.append(textNode('strong', event.op), textNode('code', `step ${event.step} · depth ${event.depth || 0} · PC ${event.pc}`));
      card.append(header);
      if (event.why) card.append(textNode('p', event.why));
      const facts = [];
      if (event.address) facts.push(`address ${event.address}`);
      if (event.error) facts.push(`error ${event.error}`);
      if (event.reverted) facts.push('frame reverted');
      if (event.gas) facts.push(`gas ${event.gas.used} used${event.gas.dynamicCost === undefined ? '' : ` · ${event.gas.dynamicCost} dynamic`}`);
      if (facts.length) card.append(textNode('code', facts.join('\n')));
      (event.storage || []).forEach(access => {
        const storage = textNode('code', `${access.transient ? 'transient ' : ''}${access.kind} ${access.slot}: ${access.before} → ${access.after}${access.appliedInFrame === false ? ' · not applied' : ''}`);
        storage.className = 'storage-fact'; card.append(storage);
      });
      events.append(card);
    });
  }
  if (evidence.links?.length) {
    links.hidden = false;
    links.append(textNode('h3', 'Causal links'));
    evidence.links.forEach(link => {
      const item = document.createElement('p');
      item.append(textNode('strong', link.kind), document.createTextNode(` · ${formatEvidenceLocation(link.from)} → ${formatEvidenceLocation(link.to)}${link.value ? ` · ${link.value}` : ''}`));
      links.append(item);
    });
  }
}

function profileLabel(profile) {
  return ({auto: 'What happened', revert: 'Failure path', storage: 'State changes', call: 'Call frames', gas: 'Gas causes', abi: 'Return data', arithmetic: 'Value flow'})[profile] || profile;
}

function formatEvidenceLocation(location) {
  return `${location.op} D${location.depth || 0}:PC${location.pc}`;
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
  const cost = comparableTraceGasCost(step) ? traceGasCost(step) : 'not compared';
  const details = document.createElement('pre'); details.textContent = `gas ${step.gasBefore} → ${step.gasAfter} · cost ${cost}\nstack pre  ${formatStack(step.stackBefore)}\nstack post ${formatStack(step.stackAfter)}${step.address ? `\naddress ${step.address}` : ''}${step.haltClass ? `\nhalt ${step.haltClass}` : ''}`;
  cell.append(title, details); return cell;
}

function stepsEqual(a, b) {
  if (!a || !b) return false;
  if (resultMode === 'transaction') {
    const identityMatches = ['depth','pc','opcode'].every(key => JSON.stringify(a[key] ?? null) === JSON.stringify(b[key] ?? null));
    return identityMatches && (!comparableTraceGasCost(a) || traceGasCost(a) === traceGasCost(b));
  }
  return ['pc','opcode','opcodeName','gasBefore','gasAfter','stackBefore','stackAfter','haltClass']
    .every(key => JSON.stringify(a[key] ?? null) === JSON.stringify(b[key] ?? null));
}
function traceGasCost(step) { return Math.max(0, step.gasBefore - step.gasAfter); }
function comparableTraceGasCost(step) { return !['CALL','CALLCODE','DELEGATECALL','STATICCALL','CREATE','CREATE2'].includes(step.opcodeName); }
function formatDivergenceField(field) { return field === 'gasCost' ? 'gas cost' : field; }
function formatStack(stack) { return stack ? `[${stack.join(', ')}]` : 'not compared'; }
function formatValue(value) { return typeof value === 'string' ? value : JSON.stringify(value, null, 2); }
function textNode(tag, text) { const node = document.createElement(tag); node.textContent = text; return node; }
function shortHash(value) { return value.length > 18 ? `${value.slice(0, 10)}…${value.slice(-6)}` : value; }
function externalLink(text, href) { const link = document.createElement('a'); link.textContent = text; link.href = href; link.target = '_blank'; link.rel = 'noreferrer'; return link; }
function badge(text) { const node = textNode('span', text); node.className = 'badge'; return node; }

async function copyCLI() {
  let command;
  if (resultMode === 'transaction') command = `echoevm verify ${lastResult.transaction.hash} --format evidence-json --profile ${lastResult.evidence?.profile || 'auto'} --limit 40`;
  else { const r = lastResult.request; command = `echoevm diff --code ${r.bytecode} --input ${r.calldata} --gas ${r.gasLimit} --format text`; }
  await navigator.clipboard.writeText(command); el('copy-cli').textContent = 'Copied'; setTimeout(() => el('copy-cli').textContent = 'Copy CLI command', 1200);
}
async function copyShareLink() {
  await navigator.clipboard.writeText(window.location.href); flashButton(el('copy-link'), 'Copied');
}
async function copyEvidence() {
  if (!lastResult?.evidence) return;
  await navigator.clipboard.writeText(JSON.stringify(lastResult.evidence, null, 2)); flashButton(el('copy-evidence'), 'Copied');
}
function flashButton(button, text) {
  const original = button.textContent; button.textContent = text; setTimeout(() => button.textContent = original, 1200);
}
function exportJSON() {
  const exported = resultMode === 'transaction' && lastResult.evidence ? lastResult.evidence : lastResult;
  const blob = new Blob([JSON.stringify(exported, null, 2) + '\n'], {type: 'application/json'}); const url = URL.createObjectURL(blob);
  const link = document.createElement('a'); link.href = url; link.download = resultMode === 'transaction' ? 'echoevm-evidence.json' : 'echoevm-differential.json'; link.click(); URL.revokeObjectURL(url);
}

function updateShareURL(hash, profile) {
  const next = `/tx/${encodeURIComponent(hash)}?profile=${encodeURIComponent(profile)}`;
  window.history.replaceState({hash, profile}, '', next);
}

function hydrateDeepLink() {
  const match = window.location.pathname.match(/^\/tx\/(0x[0-9a-fA-F]{64})\/?$/);
  if (!match) return;
  const profile = new URLSearchParams(window.location.search).get('profile');
  if (['auto','revert','storage','call','gas','abi','arithmetic'].includes(profile)) el('evidence-profile').value = profile;
  el('transaction-input').value = match[1];
  replay();
}
