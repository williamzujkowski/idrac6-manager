// dashboard.js — Dashboard view logic

const Dashboard = {
    async refresh() {
        await Promise.allSettled([
            this.refreshInfo(),
            this.refreshPower(),
            this.refreshSensors(),
            this.refreshVMedia(),
            this.refreshSEL(),
        ]);
    },

    async refreshInfo() {
        try {
            const info = await App.api('GET', App.hostPath('/info'));
            document.getElementById('info-hostname').textContent = info.hostname || '--';
            document.getElementById('info-model').textContent = info.model || '--';
            document.getElementById('info-tag').textContent = info.serviceTag || '--';
            document.getElementById('info-bios').textContent = info.biosVersion || '--';
            document.getElementById('info-fw').textContent = info.fwVersion || '--';
            document.getElementById('info-os').textContent = info.osName || '--';
        } catch (err) {
            console.error('Failed to load system info:', err);
        }
    },

    async refreshPower() {
        try {
            const power = await App.api('GET', App.hostPath('/power'));
            const indicator = document.getElementById('power-indicator');
            const stateEl = document.getElementById('power-state');

            if (power.status === 'on') {
                indicator.className = 'power-indicator on';
                stateEl.textContent = 'POWERED ON';
                stateEl.style.color = 'var(--power-on)';
            } else {
                indicator.className = 'power-indicator';
                stateEl.textContent = 'POWERED OFF';
                stateEl.style.color = 'var(--power-off)';
            }
        } catch (err) {
            console.error('Failed to load power state:', err);
        }
    },

    async refreshSensors() {
        try {
            const sensors = await App.api('GET', App.hostPath('/sensors'));
            this.renderSensors('temp-sensors', sensors.temperatures, 'C', 80);
            this.renderSensors('fan-sensors', sensors.fans, 'RPM', 15000);
            this.renderSensors('voltage-sensors', sensors.voltages, 'V', 15);
        } catch (err) {
            console.error('Failed to load sensors:', err);
        }
    },

    renderSensors(containerId, sensors, unit, maxVal) {
        const container = document.getElementById(containerId);
        if (!sensors || sensors.length === 0) {
            container.innerHTML = '<div class="empty-state">No data available</div>';
            return;
        }

        container.innerHTML = sensors.map(s => {
            const statusClass = this.sensorStatusClass(s);
            const pct = Math.min(100, (s.value / maxVal) * 100);
            return `
                <div class="sensor-item ${statusClass}">
                    <span class="sensor-name">${this.escapeHtml(s.name)}</span>
                    <span class="sensor-value">${s.value} ${unit}</span>
                    <div class="sensor-bar">
                        <div class="sensor-bar-fill" style="width:${pct}%"></div>
                    </div>
                </div>
            `;
        }).join('');
    },

    sensorStatusClass(sensor) {
        if (sensor.status === 'critical' || (sensor.critical > 0 && sensor.value >= sensor.critical)) {
            return 'sensor-crit';
        }
        if (sensor.status === 'warning' || (sensor.warning > 0 && sensor.value >= sensor.warning)) {
            return 'sensor-warn';
        }
        return 'sensor-ok';
    },

    async refreshVMedia() {
        try {
            const vm = await App.api('GET', App.hostPath('/virtualmedia'));
            const el = document.getElementById('vmedia-status');
            if (vm.url) {
                el.innerHTML = `<span>Mounted: <strong>${this.escapeHtml(vm.url)}</strong></span>`;
            } else {
                el.innerHTML = '<span>Status: Not mounted</span>';
            }
        } catch (err) {
            console.error('Failed to load virtual media status:', err);
        }
    },

    async refreshSEL() {
        try {
            const sel = await App.api('GET', App.hostPath('/sel'));
            const body = document.getElementById('sel-body');

            if (!sel.entries || sel.entries.length === 0) {
                body.innerHTML = '<tr><td colspan="4" class="empty-state">No events</td></tr>';
                return;
            }

            body.innerHTML = sel.entries.map(e => `
                <tr>
                    <td>${this.escapeHtml(e.id)}</td>
                    <td>${this.escapeHtml(e.timestamp)}</td>
                    <td class="${this.severityClass(e.severity)}">${this.escapeHtml(e.severity)}</td>
                    <td>${this.escapeHtml(e.description)}</td>
                </tr>
            `).join('');
        } catch (err) {
            console.error('Failed to load SEL:', err);
        }
    },

    severityClass(severity) {
        const s = (severity || '').toLowerCase();
        if (s.includes('critical') || s.includes('error')) return 'severity-critical';
        if (s.includes('warning') || s.includes('warn')) return 'severity-warning';
        return 'severity-normal';
    },

    escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
};
