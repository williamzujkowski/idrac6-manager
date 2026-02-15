// app.js — Core API client and state management

const App = {
    currentHost: 'default',
    refreshInterval: null,
    refreshMs: 5000,

    // API helper
    async api(method, path, body) {
        const opts = {
            method,
            headers: { 'Content-Type': 'application/json' },
        };
        if (body) opts.body = JSON.stringify(body);

        const resp = await fetch('/api' + path, opts);
        const data = await resp.json();

        if (!resp.ok) {
            throw new Error(data.error || `HTTP ${resp.status}`);
        }
        return data;
    },

    // Show connection status
    setStatus(msg, type) {
        const el = document.getElementById('connection-status');
        el.textContent = msg;
        el.className = 'status-bar ' + type;
    },

    clearStatus() {
        const el = document.getElementById('connection-status');
        el.className = 'status-bar';
    },

    // Initialize the app
    async init() {
        try {
            // Load hosts
            const hosts = await this.api('GET', '/hosts');
            if (hosts.length > 0) {
                this.currentHost = hosts[0].id;
                this.renderHostSelector(hosts);
                document.getElementById('footer-host').textContent = hosts[0].host;
            }

            // Initial data load
            await Dashboard.refresh();

            // Start auto-refresh
            this.startRefresh();

            this.clearStatus();
        } catch (err) {
            this.setStatus('Connection failed: ' + err.message, 'error');
        }
    },

    renderHostSelector(hosts) {
        const nav = document.getElementById('host-selector');
        if (hosts.length <= 1) {
            nav.textContent = hosts[0] ? hosts[0].name : '';
            return;
        }

        const select = document.createElement('select');
        hosts.forEach(h => {
            const opt = document.createElement('option');
            opt.value = h.id;
            opt.textContent = h.name + ' (' + h.host + ')';
            select.appendChild(opt);
        });
        select.onchange = () => {
            this.currentHost = select.value;
            Dashboard.refresh();
        };
        nav.appendChild(select);
    },

    startRefresh() {
        if (this.refreshInterval) clearInterval(this.refreshInterval);
        this.refreshInterval = setInterval(() => Dashboard.refresh(), this.refreshMs);
    },

    hostPath(path) {
        return '/hosts/' + this.currentHost + path;
    }
};

// Power action handler (called from HTML onclick)
async function powerAction(action) {
    const confirmActions = ['off', 'reset', 'nmi'];
    if (confirmActions.includes(action)) {
        if (!confirm('Are you sure you want to ' + action.toUpperCase() + ' the server?')) return;
    }

    try {
        await App.api('POST', App.hostPath('/power'), { action });
        App.setStatus('Power action "' + action + '" sent', 'connected');
        setTimeout(() => Dashboard.refreshPower(), 2000);
    } catch (err) {
        App.setStatus('Power action failed: ' + err.message, 'error');
    }
}

// Virtual media handlers
async function mountMedia() {
    const url = document.getElementById('vmedia-url').value.trim();
    if (!url) return;

    try {
        await App.api('POST', App.hostPath('/virtualmedia'), { url });
        App.setStatus('Mounting image...', 'connected');
        Dashboard.refreshVMedia();
    } catch (err) {
        App.setStatus('Mount failed: ' + err.message, 'error');
    }
}

async function unmountMedia() {
    try {
        await App.api('DELETE', App.hostPath('/virtualmedia'));
        App.setStatus('Image unmounted', 'connected');
        Dashboard.refreshVMedia();
    } catch (err) {
        App.setStatus('Unmount failed: ' + err.message, 'error');
    }
}

// SEL handlers
async function refreshSEL() { Dashboard.refreshSEL(); }
async function clearSEL() {
    if (!confirm('Clear all system event log entries?')) return;
    try {
        await App.api('DELETE', App.hostPath('/sel'));
        App.setStatus('Event log cleared', 'connected');
        Dashboard.refreshSEL();
    } catch (err) {
        App.setStatus('Clear SEL failed: ' + err.message, 'error');
    }
}

// Boot
document.addEventListener('DOMContentLoaded', () => App.init());
