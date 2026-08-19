/**
 * dnssecManager — Alpine.js component for DNSSEC lifecycle management (Phase 1).
 *
 * Mounted on the DNSSEC card via x-data="dnssecManager('<zone>')".
 * Communicates with the JSON endpoints under /zone/edit/:name/dnssec.
 */
function dnssecManager(zoneName) {
    return {
        zoneName,
        loaded: false,
        error: null,
        busy: false,
        enabled: false,
        keys: [],
        pendingDeleteKeyID: null,
        _deleteModal: null,
        _dsModal: null,

        async init() {
            await this.loadKeys();
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

        showDS(key) {
            const body = document.getElementById('ds-modal-body');
            if (!body) return;

            // Clear previous content and listeners
            body.replaceChildren();

            if (!key.ds || key.ds.length === 0) {
                const empty = document.createElement('p');
                empty.className = 'text-muted';
                empty.textContent = 'No DS records available for this key.';
                body.appendChild(empty);
            } else {
                const intro = document.createElement('p');
                intro.className = 'text-muted small mb-2';
                intro.textContent = 'Add these DS records to your registrar or parent zone to complete the chain of trust.';
                body.appendChild(intro);

                key.ds.forEach(ds => {
                    const group = document.createElement('div');
                    group.className = 'input-group mb-2';

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
                    body.appendChild(group);
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
