const statusEl = document.getElementById('connection-status');
const opcodeListEl = document.getElementById('opcode-list');
const stackListEl = document.getElementById('stack-list');
const stackSizeEl = document.getElementById('stack-size');
const memoryViewEl = document.getElementById('memory-view');
const runBtn = document.getElementById('run-btn');
const clearBtn = document.getElementById('clear-btn');

let socket;
let isConnected = false;

runBtn.disabled = true;

function connect() {
    statusEl.textContent = 'Connecting...';
    socket = new WebSocket('ws://' + window.location.host + '/ws');

    socket.onopen = function() {
        console.log('Connected');
        statusEl.textContent = 'Connected';
        statusEl.className = 'status connected';
        isConnected = true;
        runBtn.disabled = false;
        resetView('Ready for execution...');
    };

    socket.onclose = function() {
        console.log('Disconnected');
        statusEl.textContent = 'Disconnected';
        statusEl.className = 'status disconnected';
        isConnected = false;
        runBtn.disabled = true;
        // Try to reconnect?
        setTimeout(connect, 2000);
    };

    socket.onmessage = function(event) {
        try {
            const msg = JSON.parse(event.data);
            handleMessage(msg);
        } catch (e) {
            console.error('Invalid JSON', e);
        }
    };
}

function handleMessage(msg) {
    if (msg.type === 'step') {
        // Render step
        // msg.pre is the state BEFORE execution
        // msg.post is the state AFTER execution
        // The server sends PRE first, then POST. Or just PRE for streaming?
        // Our server implementation sends:
        // 1. PRE step (before opcode exec)
        // 2. POST step (after opcode exec)
        
        // Let's just render the PRE step for the opcode list
        if (msg.pre && !msg.post) {
            appendOpcode(msg.pre);
        }
        
        // And use either PRE or POST to update Stack/Memory lookup
        // Ideally we want to see the effect of the opcode, so POST is better for Stack/Memory updates.
        // But if we step, we want to see the state.
        
        const state = msg.post || msg.pre;
        if (state) {
            updateStack(state.stack);
        }
        
        if (msg.memory_hex) {
            updateMemory(msg.memory_hex);
        }
    } else if (msg.type === 'final') {
        appendLog(`Execution Finished. Return: ${msg.return_data_hex}`);
        if(msg.reverted) {
             appendLog(`REVERTED`, true);
        }
    } else if (msg.type === 'start') {
        resetView('Running...');
    } else {
        appendLog('Unknown message type', true);
    }
}

function appendOpcode(step) {
    // Scroll to bottom
    const div = document.createElement('div');
    div.className = 'step';
    div.innerHTML = `
        <span class="pc">[${step.pc}]</span>
        <span class="opcode">${step.opcode_name}</span>
        <span class="args"></span>
    `;
    opcodeListEl.appendChild(div);
    div.scrollIntoView({ behavior: 'smooth' });
}

function updateStack(stack) {
    stackListEl.innerHTML = '';
    if (!stack) {
        stackSizeEl.textContent = '(0)';
        return;
    }
    stackSizeEl.textContent = `(${stack.length})`;
    // Stack is usually displayed top-down (last item is top)
    // The server sends it as a list. Usually index 0 is bottom.
    // Let's reverse it to show Top at the top.
    const reversed = [...stack].reverse();
    
    reversed.forEach((val, index) => {
        const div = document.createElement('div');
        div.className = 'stack-item';
        // Calculate original index (from bottom 0)
        const originalIndex = stack.length - 1 - index;
        div.innerHTML = `
            <span class="stack-index">${originalIndex}:</span>
            <span class="stack-val">${val}</span>
        `;
        stackListEl.appendChild(div);
    });
}

function updateMemory(hexData) {
    // Hex data is a big string. Let's chunk it into 32-byte (64 char) lines.
    memoryViewEl.innerHTML = '';
    if (!hexData) return;
    
    // Remove if empty
    if (hexData.length === 0) return;

    for (let i = 0; i < hexData.length; i += 64) {
        const chunk = hexData.substr(i, 64);
        const offset = i / 2; // byte offset
        const offsetHex = offset.toString(16).padStart(4, '0');
        
        const div = document.createElement('div');
        div.className = 'memory-row';
        div.innerHTML = `
            <span class="mem-addr">0x${offsetHex}:</span>
            <span class="mem-data">${chunk}</span>
        `;
        memoryViewEl.appendChild(div);
    }
}

function appendLog(text, isError) {
    const div = document.createElement('div');
    div.className = 'step';
    if (isError) div.style.color = 'red';
    else div.style.color = '#808080';
    div.textContent = text;
    opcodeListEl.appendChild(div);
    div.scrollIntoView({ behavior: 'smooth' });
}

clearBtn.addEventListener('click', () => {
    resetView('Waiting for execution...');
});

runBtn.addEventListener('click', () => {
    if (!isConnected) {
        appendLog('Not connected', true);
    } else {
        socket.send(JSON.stringify({ type: 'run' }));
    }
});

function resetView(statusText) {
    opcodeListEl.innerHTML = '';
    stackListEl.innerHTML = '';
    memoryViewEl.innerHTML = '';
    stackSizeEl.textContent = '(0)';
    if (statusText) {
        appendLog(statusText);
    } else {
        appendLog('');
    }
}

// Start
connect();
