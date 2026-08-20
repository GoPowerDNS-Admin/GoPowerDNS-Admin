/**
 * dnssecManager — Alpine.js component for DNSSEC lifecycle management.
 *
 * Mounted on the DNSSEC card via x-data="dnssecManager('<zone>')".
 * Communicates with the JSON endpoints under /zone/edit/:name/dnssec.
 */
function dnssecManager(zoneName, initialEnabled) {
    return {
        zoneName,
        loaded: false,
        error: null,
        busy: false,
        enabled: !!initialEnabled,
        keys: [],
        pendingDeleteKeyID: null,
        _deleteModal: null,
        _dsModal: null,

        // Trust chain state
        chainBusy: false,
        chainResult: null,
        chainError: null,

        async onToggle() {
            // Load on first expand only; subsequent toggles reuse cached state.
            if (!this.loaded) {
                await this.loadKeys();
            }
        },

        baseUrl() {
            return `/zone/edit/${this.zoneName}/dnssec`;
        },

        async loadKeys() {
            this.loaded = false;
            this.error = null;
            try {
                const r = await fetch(this.baseUrl());
                const data = await r.json();
                if (!data.success) throw new Error(data.message || 'Unknown error');
                this.keys = data.keys || [];
                this.enabled = this.keys.length > 0;
            } catch (e) {
                this.error = 'Failed to load DNSSEC status: ' + e.message;
            } finally {
                this.loaded = true;
            }
        },

        async enableDNSSEC() {
            if (!confirm('Enable DNSSEC for this zone? A Combined Signing Key (CSK) will be generated automatically.')) return;
            this.busy = true;
            try {
                const r = await fetch(`${this.baseUrl()}/enable`, { method: 'POST' });
                const data = await r.json();
                if (!data.success) throw new Error(data.message || 'Unknown error');
                showToast('DNSSEC enabled successfully.', 'success');
                await this.loadKeys();
            } catch (e) {
                showToast('Failed to enable DNSSEC: ' + e.message, 'danger');
            } finally {
                this.busy = false;
            }
        },

        async disableDNSSEC() {
            if (!confirm('Disable DNSSEC? All cryptokeys will be deleted. Ensure DS records are removed from the parent zone first.')) return;
            this.busy = true;
            try {
                const r = await fetch(`${this.baseUrl()}/disable`, { method: 'POST' });
                const data = await r.json();
                if (!data.success) throw new Error(data.message || 'Unknown error');
                showToast('DNSSEC disabled.', 'success');
                await this.loadKeys();
            } catch (e) {
                showToast('Failed to disable DNSSEC: ' + e.message, 'danger');
            } finally {
                this.busy = false;
            }
        },

        async toggleActive(key) {
            const newState = !key.active;
            const label = newState ? 'activate' : 'deactivate';
            if (!confirm(`${label.charAt(0).toUpperCase() + label.slice(1)} key ${key.id}?`)) return;
            try {
                const r = await fetch(
                    `${this.baseUrl()}/keys/${key.id}/toggle`,
                    {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ active: newState }),
                    }
                );
                const data = await r.json();
                if (!data.success) throw new Error(data.message || 'Unknown error');
                showToast(`Key ${key.id} ${label}d.`, 'success');
                await this.loadKeys();
            } catch (e) {
                showToast(`Failed to ${label} key: ` + e.message, 'danger');
            }
        },

        async togglePublished(key) {
            const newState = !key.published;
            const label = newState ? 'publish' : 'unpublish';
            if (!confirm(`${label.charAt(0).toUpperCase() + label.slice(1)} key ${key.id}?`)) return;
            try {
                const r = await fetch(
                    `${this.baseUrl()}/keys/${key.id}/toggle`,
                    {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ published: newState }),
                    }
                );
                const data = await r.json();
                if (!data.success) throw new Error(data.message || 'Unknown error');
                showToast(`Key ${key.id} ${label}ed.`, 'success');
                await this.loadKeys();
            } catch (e) {
                showToast(`Failed to ${label} key: ` + e.message, 'danger');
            }
        },

        confirmDeleteKey(key) {
            this.pendingDeleteKeyID = key.id;
            if (!this._deleteModal) {
                this._deleteModal = new bootstrap.Modal(document.getElementById('deleteKeyModal'));
            }
            this._deleteModal.show();
        },

        async deleteKey() {
            const id = this.pendingDeleteKeyID;
            if (!id) return;
            if (this._deleteModal) this._deleteModal.hide();
            try {
                const r = await fetch(
                    `${this.baseUrl()}/keys/${id}/delete`,
                    { method: 'POST' }
                );
                const data = await r.json();
                if (!data.success) throw new Error(data.message || 'Unknown error');
                showToast(`Key ${id} deleted.`, 'success');
                this.pendingDeleteKeyID = null;
                await this.loadKeys();
            } catch (e) {
                showToast('Failed to delete key: ' + e.message, 'danger');
            }
        },

        async checkChain() {
            this.chainBusy = true;
            this.chainResult = null;
            this.chainError = null;
            try {
                const r = await fetch(`${this.baseUrl()}/chain`);
                const data = await r.json();
                if (!data.success) throw new Error(data.message || 'Unknown error');
                this.chainResult = data.chain;
            } catch (e) {
                this.chainError = 'Chain check failed: ' + e.message;
            } finally {
                this.chainBusy = false;
            }
        },

        showDS(key) {
            const body = document.getElementById('ds-modal-body');
            if (!body) return;

            body.replaceChildren();

            if (!key.ds || key.ds.length === 0) {
                const empty = document.createElement('p');
                empty.className = 'text-muted';
                empty.textContent = 'No DS records available for this key.';
                body.appendChild(empty);
            } else {
                // Best practices info box
                const info = document.createElement('div');
                info.className = 'alert alert-info small mb-3';
                info.innerHTML =
                    '<strong><i class="bi bi-info-circle me-1"></i>Best practice:</strong> ' +
                    'Submit only the <strong>SHA-256 (type 2)</strong> DS record to your registrar. ' +
                    'SHA-1 (type 1) is considered weak and most registries accept or prefer SHA-256 only. ' +
                    'Submitting both is harmless but unnecessary. ' +
                    'Never submit SHA-384 (type 4) unless explicitly required by your registry.';
                body.appendChild(info);

                key.ds.forEach(ds => {
                    // DS format from PowerDNS: "<keytag> <algo> <digesttype> <digest>"
                    // If the string contains "DS" (full zone-file format), find the offset;
                    // otherwise treat it as the raw 4-field form.
                    const parts = ds.trim().split(/\s+/);
                    const dsIdx = parts.findIndex(p => p.toUpperCase() === 'DS');
                    const base = dsIdx !== -1 ? dsIdx + 1 : 0;
                    const digestType = parseInt(parts[base + 2], 10);

                    const digestInfo = dsDigestInfo(digestType);

                    const wrapper = document.createElement('div');
                    wrapper.className = 'mb-3';

                    // Label row
                    const labelRow = document.createElement('div');
                    labelRow.className = 'd-flex align-items-center gap-2 mb-1';

                    const badge = document.createElement('span');
                    badge.className = `badge ${digestInfo.badgeClass}`;
                    badge.textContent = digestInfo.label;
                    labelRow.appendChild(badge);

                    const desc = document.createElement('span');
                    desc.className = 'small text-muted';
                    desc.textContent = digestInfo.description;
                    labelRow.appendChild(desc);

                    wrapper.appendChild(labelRow);

                    // Input + copy button
                    const group = document.createElement('div');
                    group.className = 'input-group';

                    const input = document.createElement('input');
                    input.type = 'text';
                    input.className = 'form-control form-control-sm font-monospace';
                    input.value = ds;
                    input.readOnly = true;

                    const btn = document.createElement('button');
                    btn.type = 'button';
                    btn.className = 'btn btn-sm btn-outline-secondary';
                    btn.innerHTML = '<i class="bi bi-clipboard"></i>';
                    btn.addEventListener('click', () => {
                        navigator.clipboard.writeText(ds).then(() => {
                            showToast('Copied!', 'success');
                        }).catch(() => {
                            showToast('Copy failed — select and copy manually.', 'warning');
                        });
                    });

                    group.appendChild(input);
                    group.appendChild(btn);
                    wrapper.appendChild(group);
                    body.appendChild(wrapper);
                });

                const meta = document.createElement('p');
                meta.className = 'text-muted small mt-2 mb-0';
                meta.innerHTML = `Key ID: <strong>${key.id}</strong> &mdash; Type: <strong class="text-uppercase">${escapeHtml(key.key_type)}</strong> &mdash; Algorithm: <strong>${escapeHtml(key.algorithm)}</strong>`;
                body.appendChild(meta);
            }

            if (!this._dsModal) {
                this._dsModal = new bootstrap.Modal(document.getElementById('dsModal'));
            }
            this._dsModal.show();
        },
    };
}

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

// dsDigestInfo returns display metadata for a DS digest type number (RFC 4034 / IANA registry).
function dsDigestInfo(type) {
    switch (type) {
        case 1:
            return {
                label: 'Type 1 — SHA-1',
                badgeClass: 'bg-warning text-dark',
                description: 'Deprecated — weak, avoid submitting to registrar',
            };
        case 2:
            return {
                label: 'Type 2 — SHA-256',
                badgeClass: 'bg-success',
                description: 'Recommended — submit this one to your registrar',
            };
        case 3:
            return {
                label: 'Type 3 — GOST R 34.11-94',
                badgeClass: 'bg-secondary',
                description: 'Obsolete — do not use',
            };
        case 4:
            return {
                label: 'Type 4 — SHA-384',
                badgeClass: 'bg-info text-dark',
                description: 'Stronger than SHA-256, but rarely required',
            };
        default:
            return {
                label: `Type ${isNaN(type) ? '?' : type}`,
                badgeClass: 'bg-secondary',
                description: 'Unknown digest type',
            };
    }
}
