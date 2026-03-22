/**
 * Zone Edit — Alpine.js component
 *
 * Pure utility functions (showToast, showConfirm, DNS helpers) come first and
 * have no dependency on Alpine. The Alpine component factory `zoneEditor()`
 * follows and is registered both as a global and via Alpine.data so that it can be
 * invoked as  x-data="zoneEditor()"  regardless of script-load order.
 * Init data is read from <script type="application/json" id="zone-data">.
 */

// ── Toast helper ──────────────────────────────────────────────────────────────

function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const configs = {
        success: { icon: 'bi-check-circle-fill',        bg: 'bg-success', title: 'Success' },
        danger:  { icon: 'bi-exclamation-triangle-fill', bg: 'bg-danger',  title: 'Error'   },
        warning: { icon: 'bi-exclamation-circle-fill',   bg: 'bg-warning', title: 'Warning' },
        info:    { icon: 'bi-info-circle-fill',           bg: 'bg-info',    title: 'Info'    },
    };
    const cfg = configs[type] || configs.info;
    const id = `toast-${Date.now()}`;

    container.insertAdjacentHTML('beforeend', `
        <div id="${id}" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
            <div class="toast-header ${cfg.bg} text-white">
                <i class="bi ${cfg.icon} me-2"></i>
                <strong class="me-auto">${cfg.title}</strong>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="toast"></button>
            </div>
            <div class="toast-body">${message}</div>
        </div>`);

    const el = document.getElementById(id);
    new bootstrap.Toast(el, { autohide: true, delay: 5000 }).show();
    el.addEventListener('hidden.bs.toast', () => el.remove());
}

// ── Confirm dialog ────────────────────────────────────────────────────────────

function showConfirm(message, opts = {}) {
    const { confirmText = 'Confirm', cancelText = 'Cancel', confirmBtnClass = 'btn-danger' } = opts;

    let modalEl = document.getElementById('genericConfirmModal');
    if (!modalEl) {
        document.body.insertAdjacentHTML('beforeend', `
            <div class="modal fade" id="genericConfirmModal" tabindex="-1" aria-hidden="true">
              <div class="modal-dialog modal-dialog-centered">
                <div class="modal-content">
                  <div class="modal-header">
                    <h5 class="modal-title">Please Confirm</h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                  </div>
                  <div class="modal-body"><p id="genericConfirmMessage" class="mb-0"></p></div>
                  <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" id="genericConfirmCancelBtn">Cancel</button>
                    <button type="button" class="btn btn-danger"    id="genericConfirmOkBtn">Confirm</button>
                  </div>
                </div>
              </div>
            </div>`);
        modalEl = document.getElementById('genericConfirmModal');
    }

    modalEl.querySelector('#genericConfirmMessage').textContent = message;
    const cancelBtn = modalEl.querySelector('#genericConfirmCancelBtn');
    const okBtn     = modalEl.querySelector('#genericConfirmOkBtn');
    cancelBtn.textContent = cancelText;
    okBtn.textContent     = confirmText;
    okBtn.className       = 'btn ' + confirmBtnClass;

    return new Promise(resolve => {
        const bsModal = bootstrap.Modal.getOrCreateInstance(modalEl);

        function cleanup() {
            okBtn.removeEventListener('click', onOk);
            cancelBtn.removeEventListener('click', onCancel);
            modalEl.removeEventListener('hidden.bs.modal', onHidden);
        }
        function onOk()     { cleanup(); resolve(true);  bsModal.hide(); }
        function onCancel() { cleanup(); resolve(false); }
        function onHidden() { cleanup(); resolve(false); }

        okBtn.addEventListener('click', onOk, { once: true });
        cancelBtn.addEventListener('click', onCancel, { once: true });
        modalEl.addEventListener('hidden.bs.modal', onHidden, { once: true });
        bsModal.show();
    });
}

// ── DNS utility functions ─────────────────────────────────────────────────────

function canonicalizeHostname(hostname) {
    if (!hostname) return hostname;
    hostname = hostname.trim();
    if (hostname.endsWith('.')) return hostname;
    return hostname + '.';
}

function parseSOA(content) {
    if (!content) return null;
    const parts = content.trim().split(/\s+/);
    if (parts.length < 7) return null;
    return { mname: parts[0], rname: parts[1], serial: parts[2],
             refresh: parts[3], retry: parts[4], expire: parts[5], minimum: parts[6] };
}

function composeSOA(fields) {
    const intKeys = ['serial', 'refresh', 'retry', 'expire', 'minimum'];
    for (const k of intKeys) {
        if (fields[k] === '' || fields[k] == null) return null;
        const n = Number.parseInt(String(fields[k]), 10);
        if (!Number.isFinite(n) || n < 0) return null;
        fields[k] = String(n);
    }
    const mname = canonicalizeHostname(fields.mname || '');
    const rname = canonicalizeHostname(fields.rname || '');
    if (!mname || !rname) return null;
    return `${mname} ${rname} ${fields.serial} ${fields.refresh} ${fields.retry} ${fields.expire} ${fields.minimum}`;
}

function isValidIPv4(ip) {
    const parts = ip.trim().split('.');
    if (parts.length !== 4) return false;
    return parts.every(p => /^\d+$/.test(p) && Number(p) >= 0 && Number(p) <= 255);
}

function isValidIPv6(ip) {
    ip = ip.trim();
    try {
        const bare = ip.startsWith('[') && ip.endsWith(']') ? ip.slice(1, -1) : ip;
        const url = new URL('http://[' + bare + ']');
        return url.hostname === '[' + bare + ']';
    } catch (_) { /* fall through to regex */ }
    return /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]+|::(ffff(:0{1,4})?:)?((25[0-5]|(2[0-4]|1?[0-9])?[0-9])\.){3}(25[0-5]|(2[0-4]|1?[0-9])?[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1?[0-9])?[0-9])\.){3}(25[0-5]|(2[0-4]|1?[0-9])?[0-9]))$/.test(ip);
}

function parseMX(content) {
    if (!content) return null;
    const parts = content.trim().split(/\s+/);
    if (parts.length < 2) return null;
    const priority = Number.parseInt(parts[0], 10);
    if (!Number.isFinite(priority) || priority < 0 || priority > 65535) return null;
    return { priority: String(priority), hostname: parts.slice(1).join(' ') };
}

function composeMX(fields) {
    const priority = Number.parseInt(String(fields.priority), 10);
    if (!Number.isFinite(priority) || priority < 0 || priority > 65535) return null;
    const hostname = canonicalizeHostname((fields.hostname || '').trim());
    if (!hostname) return null;
    return `${priority} ${hostname}`;
}

/**
 * Parse a TXT record content string (zone-file quoted format) into plain text.
 * Handles multiple quoted segments, which DNS concatenates without separator.
 * E.g. `"v=spf1" " include:example.com"` → `"v=spf1 include:example.com"`
 */
function parseTXT(content) {
    if (!content) return '';
    const trimmed = content.trim();
    const parts = [];
    const re = /"((?:[^"\\]|\\.)*)"/g;
    let match;
    while ((match = re.exec(trimmed)) !== null) {
        parts.push(match[1].replace(/\\"/g, '"').replace(/\\\\/g, '\\'));
    }
    if (parts.length > 0) return parts.join('');
    // Not in quoted format — return as-is (best-effort).
    return trimmed;
}

/**
 * Compose plain text back into zone-file TXT content.
 * Strings longer than 255 chars are split into 255-char quoted chunks (DNS limit per string).
 */
function composeTXT(text) {
    if (text == null) return null;
    text = String(text);
    if (text.length === 0) return '""';
    function escapeChunk(s) {
        return '"' + s.replace(/\\/g, '\\\\').replace(/"/g, '\\"') + '"';
    }
    if (text.length <= 255) return escapeChunk(text);
    const chunks = [];
    for (let i = 0; i < text.length; i += 255) {
        chunks.push(escapeChunk(text.slice(i, i + 255)));
    }
    return chunks.join(' ');
}

function canonicalizeContent(type, content) {
    if (!content) return content;
    content = content.trim();
    const fqdnTypes = ['CNAME', 'MX', 'NS', 'PTR', 'SRV'];
    if (!fqdnTypes.includes(type)) return content;
    if (type === 'MX') {
        const parts = content.split(/\s+/);
        if (parts.length >= 2) return `${parts[0]} ${canonicalizeHostname(parts.slice(1).join(' '))}`;
    } else if (type === 'SRV') {
        const parts = content.split(/\s+/);
        if (parts.length >= 4) return `${parts[0]} ${parts[1]} ${parts[2]} ${canonicalizeHostname(parts.slice(3).join(' '))}`;
    } else {
        return canonicalizeHostname(content);
    }
    return content;
}

// ── Alpine component factory ──────────────────────────────────────────────────

function zoneEditor(initData) {
    if (!initData) {
        const el = document.getElementById('zone-data');
        initData = el ? JSON.parse(el.textContent) : {};
    }
    return {
        // ── Initialisation data ───────────────────────────────────────────────
        zoneName:     initData.zoneName     || '',
        allowedTypes: initData.allowedTypes || [],
        records:      (initData.records || []).map(r => ({ ...r })),
        ttlPresets:   initData.ttlPresets   || [],

        // Set once in init() from the server-provided snapshot — never mutated.
        _originalKeys: {},  // { 'name|type': true }
        _initialTypeSet: {}, // { 'A': true, ... } — for "new type" badge

        // ── Pending changes (plain object for Alpine reactivity) ──────────────
        // Keys: 'name|type'. Values: RecordChange-shaped objects.
        pendingChanges: {},

        // ── Filter / sort / pagination ────────────────────────────────────────
        searchQuery:      '',
        activeTypeFilter: 'all',
        sortField:        'name',
        sortAsc:          true,
        currentPage:      1,
        pageSize:         initData.pageSize || 25,

        // ── Save state ────────────────────────────────────────────────────────
        isSaving: false,

        // ── Record modal ──────────────────────────────────────────────────────
        recordForm: {
            isEditing:       false,
            originalId:      '',
            originalName:    '',
            originalType:    '',
            originalContent: '',
            name:            '',
            type:            '',
            ttl:             3600,
            ttlPreset:       'custom',
            content:         '',
            comment:         '',
            disabled:        false,
            // MX-specific
            mxPriority: '10',
            mxHostname: '',
            // TXT-specific
            txtText: '',
        },

        // ── SOA modal ─────────────────────────────────────────────────────────
        soaForm: {
            originalId:      '',
            originalName:    '',
            originalContent: '',
            origTtl:         0,
            origDisabled:    false,
            origComment:     '',
            mname:  '',
            rname:  '',
            serial:  '',
            refresh: '',
            retry:   '',
            expire:  '',
            minimum: '',
        },

        // ── Alpine lifecycle ──────────────────────────────────────────────────

        init() {
            // Build immutable snapshots from the initial server-rendered state.
            this._originalKeys    = Object.fromEntries(this.records.map(r => [r.name + '|' + r.type, true]));
            this._initialTypeSet  = Object.fromEntries(this.records.map(r => [r.type, true]));

            // Reset page to 1 whenever search or type filter changes.
            this.$watch('searchQuery',      () => { this.currentPage = 1; });
            this.$watch('activeTypeFilter', () => { this.currentPage = 1; });

            // Fix Bootstrap aria-hidden focus-trap warning: blur any focused descendant on hide.
            ['recordModal', 'soaModal'].forEach(id => {
                const modal = document.getElementById(id);
                if (modal) {
                    modal.addEventListener('hide.bs.modal', () => {
                        if (modal.contains(document.activeElement)) document.activeElement.blur();
                    });
                }
            });
        },

        // ── Computed ──────────────────────────────────────────────────────────

        get pendingCount() {
            return Object.keys(this.pendingChanges).length;
        },

        /** All record types currently present in the table, in a consistent order. */
        get availableTypes() {
            const types = new Set(this.records.map(r => r.type));
            const priority = ['SOA', 'NS', 'A', 'AAAA', 'CNAME', 'MX', 'TXT', 'SRV', 'CAA', 'PTR'];
            return [...types].sort((a, b) => {
                const ia = priority.indexOf(a), ib = priority.indexOf(b);
                if (ia !== -1 && ib !== -1) return ia - ib;
                if (ia !== -1) return -1;
                if (ib !== -1) return 1;
                return a.localeCompare(b);
            });
        },

        /** Filtered + sorted record list (all pages). */
        get filteredRecords() {
            let list = this.records;

            if (this.activeTypeFilter !== 'all') {
                list = list.filter(r => r.type === this.activeTypeFilter);
            }

            if (this.searchQuery) {
                const q = this.searchQuery.toLowerCase();
                list = list.filter(r =>
                    r.display_name.toLowerCase().includes(q) ||
                    r.content.toLowerCase().includes(q) ||
                    (r.comment || '').toLowerCase().includes(q)
                );
            }

            return [...list].sort((a, b) => {
                let av, bv;
                switch (this.sortField) {
                    case 'type': av = a.type;              bv = b.type;              break;
                    case 'ttl':  av = a.ttl;               bv = b.ttl;               break;
                    default: {
                        // @ (zone apex) always sorts first regardless of direction.
                        const aApex = a.display_name === '@';
                        const bApex = b.display_name === '@';
                        if (aApex && !bApex) return -1;
                        if (!aApex && bApex) return  1;
                        av = a.display_name.toLowerCase();
                        bv = b.display_name.toLowerCase();
                    }
                }
                if (av < bv) return this.sortAsc ? -1 : 1;
                if (av > bv) return this.sortAsc ?  1 : -1;
                return 0;
            });
        },

        get totalPages() {
            return Math.max(1, Math.ceil(this.filteredRecords.length / this.pageSize));
        },

        get paginatedRecords() {
            const page  = Math.min(this.currentPage, this.totalPages);
            const start = (page - 1) * this.pageSize;
            return this.filteredRecords.slice(start, start + this.pageSize);
        },

        /** Help text for the Data field in the record modal, derived from the selected type. */
        get recordContentHelp() {
            const found = this.allowedTypes.find(t => t.type === this.recordForm.type);
            return (found && found.help)
                ? found.help
                : 'Record data (e.g., IP address, hostname, text). For CNAME, MX, NS, PTR, SRV records a trailing dot is added automatically.';
        },

        // ── Helpers ───────────────────────────────────────────────────────────

        recordId(r) {
            return r.name + '|' + r.type + '|' + r.content;
        },

        recordRowClass(r) {
            const key = r.name + '|' + r.type;
            if (!(key in this.pendingChanges)) return '';
            return (key in this._originalKeys) ? 'table-warning' : 'table-success';
        },

        isNewType(type) {
            return !(type in this._initialTypeSet);
        },

        getDisplayName(fullName) {
            if (!fullName) return fullName;
            const z = this.zoneName.replace(/\.$/, '');
            if (fullName === this.zoneName || fullName === z) return '@';
            if (fullName.endsWith('.' + z + '.')) return fullName.slice(0, -(z.length + 2));
            if (fullName.endsWith('.' + z))       return fullName.slice(0, -(z.length + 1));
            return fullName;
        },

        canonicalizeName(name) {
            if (!name) return name;
            name = name.trim();
            if (name === '@') return this.zoneName;
            if (!name.endsWith('.')) {
                const z = this.zoneName.replace(/\.$/, '');
                name = name.endsWith(z) ? name + '.' : name + '.' + this.zoneName;
            }
            return name;
        },

        /** Returns { content, disabled } pairs for all records matching name+type, optionally excluding one content value. */
        collectRRsetRecords(name, type, excludeContent) {
            return this.records
                .filter(r => r.name === name && r.type === type &&
                    (excludeContent === undefined || r.content !== excludeContent))
                .map(r => ({ content: r.content, disabled: r.disabled }));
        },

        // ── Sort / filter / pagination ────────────────────────────────────────

        toggleSort(field) {
            if (this.sortField === field) {
                this.sortAsc = !this.sortAsc;
            } else {
                this.sortField = field;
                this.sortAsc = true;
            }
            this.currentPage = 1;
        },

        setTypeFilter(type) {
            this.activeTypeFilter = type;
        },

        changePageSize(size) {
            this.pageSize = Number(size);
            this.currentPage = 1;
            fetch('/profile/preferences', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ zone_edit_page_size: this.pageSize }),
            });
        },

        // ── Bootstrap modal helpers ───────────────────────────────────────────

        _showModal(id) {
            const el = document.getElementById(id);
            if (el) bootstrap.Modal.getOrCreateInstance(el).show();
        },

        _hideModal(id) {
            const el = document.getElementById(id);
            if (el) bootstrap.Modal.getOrCreateInstance(el).hide();
        },

        /** Return the ttlPreset value ('custom' or a seconds string) for a given TTL number. */
        _ttlPresetFor(ttl) {
            const match = this.ttlPresets.find(p => p.seconds === ttl);
            return match ? String(match.seconds) : 'custom';
        },

        /** Sync recordForm.ttl when the preset select changes. */
        onTTLPresetChange() {
            if (this.recordForm.ttlPreset !== 'custom') {
                this.recordForm.ttl = Number(this.recordForm.ttlPreset);
            }
        },

        // ── Open record modal (add) ───────────────────────────────────────────

        openAddRecord() {
            const defaultType = this.allowedTypes.length > 0 ? this.allowedTypes[0].type : 'A';
            const defaultTTL  = this.ttlPresets.length > 0 ? this.ttlPresets[0].seconds : 3600;
            this.recordForm = {
                isEditing: false,
                originalId: '', originalName: '', originalType: '', originalContent: '',
                name: '', type: defaultType,
                ttl: defaultTTL, ttlPreset: this._ttlPresetFor(defaultTTL),
                content: '', comment: '', disabled: false,
                mxPriority: '10', mxHostname: '', txtText: '',
            };
            this._showModal('recordModal');
        },

        // ── Open record modal (edit) ──────────────────────────────────────────

        openEditRecord(record) {
            if (record.type === 'SOA') { this.openSOAModal(record); return; }
            const mx  = record.type === 'MX'  ? parseMX(record.content)  : null;
            const txt = record.type === 'TXT' ? parseTXT(record.content) : '';
            this.recordForm = {
                isEditing:       true,
                originalId:      this.recordId(record),
                originalName:    record.name,
                originalType:    record.type,
                originalContent: record.content,
                name:            record.display_name,
                type:            record.type,
                ttl:             record.ttl,
                ttlPreset:       this._ttlPresetFor(record.ttl),
                content:         record.content,
                comment:         record.comment || '',
                disabled:        record.disabled,
                mxPriority: mx?.priority || '10',
                mxHostname:  mx?.hostname  || '',
                txtText:     txt,
            };
            this._showModal('recordModal');
        },

        // ── Open SOA modal ────────────────────────────────────────────────────

        openSOAModal(record) {
            const soa = parseSOA(record.content);
            if (!soa) showToast('Malformed SOA content; cannot parse existing values.', 'danger');
            this.soaForm = {
                originalId:      this.recordId(record),
                originalName:    record.name,
                originalContent: record.content,
                origTtl:         record.ttl,
                origDisabled:    record.disabled,
                origComment:     record.comment || '',
                mname:   soa?.mname   || '',
                rname:   soa?.rname   || '',
                serial:  soa?.serial  || '',
                refresh: soa?.refresh || '',
                retry:   soa?.retry   || '',
                expire:  soa?.expire  || '',
                minimum: soa?.minimum || '',
            };
            this._showModal('soaModal');
        },

        // ── Save record modal ─────────────────────────────────────────────────

        saveRecord() {
            const form = document.getElementById('record-form');
            if (!form.checkValidity()) { form.reportValidity(); return; }

            const rf = this.recordForm;
            let content;

            if (rf.type === 'MX') {
                content = composeMX({ priority: rf.mxPriority, hostname: rf.mxHostname });
                if (!content) { showToast('Please provide a valid priority (0–65535) and hostname.', 'danger'); return; }
            } else if (rf.type === 'TXT') {
                content = composeTXT(rf.txtText);
                if (content === null) { showToast('Please provide valid TXT content.', 'danger'); return; }
            } else {
                const rawContent = (rf.content || '').trim();
                if (rf.type === 'A'    && !isValidIPv4(rawContent)) { showToast('Invalid IPv4 address for A record.', 'danger');    return; }
                if (rf.type === 'AAAA' && !isValidIPv6(rawContent)) { showToast('Invalid IPv6 address for AAAA record.', 'danger'); return; }
                content = canonicalizeContent(rf.type, rawContent);
            }

            const record = {
                name:         this.canonicalizeName(rf.name),
                type:         rf.type,
                ttl:          Number(rf.ttl),
                content:      content,
                disabled:     !!rf.disabled,
                comment:      (rf.comment || '').trim(),
                display_name: '',
            };
            record.display_name = this.getDisplayName(record.name);

            // Detect no-op edits.
            if (rf.isEditing) {
                const orig = this.records.find(r => this.recordId(r) === rf.originalId);
                if (orig &&
                    orig.name    === record.name    &&
                    orig.type    === record.type    &&
                    orig.content === record.content &&
                    orig.ttl     === record.ttl     &&
                    orig.disabled === record.disabled &&
                    (orig.comment || '') === record.comment
                ) {
                    showToast('No changes detected for this record.', 'info');
                    this._hideModal('recordModal');
                    return;
                }
            }

            // Handle name/type change — update the old RRset in pendingChanges.
            if (rf.isEditing && (rf.originalName !== record.name || rf.originalType !== record.type)) {
                const oldKey = rf.originalName + '|' + rf.originalType;
                if (oldKey in this._originalKeys) {
                    const siblings = this.collectRRsetRecords(rf.originalName, rf.originalType, rf.originalContent);
                    if (oldKey in this.pendingChanges) {
                        const updated = { ...this.pendingChanges[oldKey] };
                        updated.records = updated.records.filter(r => r.content !== rf.originalContent);
                        this.pendingChanges = { ...this.pendingChanges, [oldKey]: updated };
                    } else {
                        const oldTTL = (this.records.find(r => r.name === rf.originalName && r.type === rf.originalType)?.ttl) || 0;
                        this.pendingChanges = { ...this.pendingChanges, [oldKey]: {
                            name: rf.originalName, type: rf.originalType,
                            ttl: oldTTL, comment: '', records: siblings,
                            existed: true, changed: true,
                        }};
                    }
                } else if (oldKey in this.pendingChanges) {
                    const updated = { ...this.pendingChanges[oldKey] };
                    updated.records = updated.records.filter(r => r.content !== rf.originalContent);
                    if (updated.records.length === 0) {
                        const { [oldKey]: _, ...rest } = this.pendingChanges;
                        this.pendingChanges = rest;
                    } else {
                        this.pendingChanges = { ...this.pendingChanges, [oldKey]: updated };
                    }
                }
            }

            // Upsert the new key in pendingChanges.
            const key        = record.name + '|' + record.type;
            const existedInDB = key in this._originalKeys;

            if (!(key in this.pendingChanges)) {
                const existingRecs = existedInDB ? this.collectRRsetRecords(record.name, record.type) : [];
                this.pendingChanges = { ...this.pendingChanges, [key]: {
                    name: record.name, type: record.type,
                    ttl: record.ttl, comment: record.comment,
                    records: existingRecs, existed: existedInDB, changed: true,
                }};
            }

            const change = { ...this.pendingChanges[key] };
            change.changed = true;
            change.ttl     = record.ttl;
            change.comment = record.comment;

            // If only content changed, remove the stale content entry.
            if (rf.isEditing &&
                rf.originalName === record.name &&
                rf.originalType === record.type &&
                rf.originalContent !== record.content
            ) {
                change.records = change.records.filter(r => r.content !== rf.originalContent);
            }

            const existingIdx = change.records.findIndex(r => r.content === record.content);
            if (existingIdx >= 0) {
                change.records = change.records.map((r, i) =>
                    i === existingIdx ? { content: record.content, disabled: record.disabled } : r
                );
            } else {
                change.records = [...change.records, { content: record.content, disabled: record.disabled }];
            }

            this.pendingChanges = { ...this.pendingChanges, [key]: change };

            // Update the records array (Alpine x-for re-renders on splice).
            const origIdx = rf.isEditing
                ? this.records.findIndex(r => this.recordId(r) === rf.originalId)
                : -1;

            if (origIdx >= 0) {
                this.records.splice(origIdx, 1, record);
            } else {
                this.records.push(record);
            }

            this._hideModal('recordModal');
        },

        // ── Save SOA modal ────────────────────────────────────────────────────

        saveSOA() {
            const sf = this.soaForm;
            const composed = composeSOA({
                mname: sf.mname.trim(), rname: sf.rname.trim(),
                serial: sf.serial.trim(), refresh: sf.refresh.trim(),
                retry: sf.retry.trim(), expire: sf.expire.trim(), minimum: sf.minimum.trim(),
            });

            if (!composed) { showToast('Please provide valid SOA values.', 'danger'); return; }

            if (composed === sf.originalContent) {
                showToast('No changes detected for this SOA record.', 'info');
                this._hideModal('soaModal');
                return;
            }

            const record = {
                name:         sf.originalName,
                type:         'SOA',
                ttl:          sf.origTtl,
                content:      composed,
                disabled:     sf.origDisabled,
                comment:      sf.origComment,
                display_name: this.getDisplayName(sf.originalName),
            };

            const key = record.name + '|SOA';
            const existedInDB = key in this._originalKeys;

            if (!(key in this.pendingChanges)) {
                this.pendingChanges = { ...this.pendingChanges, [key]: {
                    name: record.name, type: 'SOA',
                    ttl: record.ttl, comment: record.comment,
                    records: [], existed: existedInDB, changed: true,
                }};
            }

            const change = { ...this.pendingChanges[key] };
            change.changed = true;
            change.ttl     = record.ttl;
            change.comment = record.comment;
            change.records = [{ content: record.content, disabled: record.disabled }];
            this.pendingChanges = { ...this.pendingChanges, [key]: change };

            const origIdx = this.records.findIndex(r => this.recordId(r) === sf.originalId);
            if (origIdx >= 0) {
                this.records.splice(origIdx, 1, record);
            } else {
                this.records.push(record);
            }

            this._hideModal('soaModal');
        },

        // ── Delete record ─────────────────────────────────────────────────────

        async deleteRecord(record) {
            const confirmed = await showConfirm('Are you sure you want to delete this record?', {
                confirmText: 'Delete', confirmBtnClass: 'btn-danger',
            });
            if (!confirmed) return;

            const key = record.name + '|' + record.type;

            if (key in this._originalKeys) {
                // Existed in DB — mark for deletion by setting remaining siblings.
                if (key in this.pendingChanges) {
                    const updated = { ...this.pendingChanges[key] };
                    updated.records = updated.records.filter(r => r.content !== record.content);
                    updated.changed = true;
                    this.pendingChanges = { ...this.pendingChanges, [key]: updated };
                } else {
                    this.pendingChanges = { ...this.pendingChanges, [key]: {
                        name: record.name, type: record.type, ttl: record.ttl, comment: '',
                        records: this.collectRRsetRecords(record.name, record.type, record.content),
                        existed: true, changed: true,
                    }};
                }
            } else {
                // Added in this session — clean up from pendingChanges.
                if (key in this.pendingChanges) {
                    const updated = { ...this.pendingChanges[key] };
                    updated.records = updated.records.filter(r => r.content !== record.content);
                    if (updated.records.length === 0) {
                        const { [key]: _, ...rest } = this.pendingChanges;
                        this.pendingChanges = rest;
                    } else {
                        this.pendingChanges = { ...this.pendingChanges, [key]: updated };
                    }
                }
            }

            const idx = this.records.findIndex(r => this.recordId(r) === this.recordId(record));
            if (idx >= 0) this.records.splice(idx, 1);
        },

        // ── Save all changes ──────────────────────────────────────────────────

        async saveChanges() {
            if (this.pendingCount === 0) { showToast('No changes to save', 'warning'); return; }

            this.isSaving = true;
            try {
                const res = await fetch(`/zone/edit/${this.zoneName}/records`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ changes: Object.values(this.pendingChanges) }),
                });

                let data;
                try { data = await res.json(); } catch (_) { data = {}; }

                if (res.ok && data.success) {
                    showToast('Records saved successfully!', 'success');
                    setTimeout(() => location.reload(), 1000);
                } else {
                    showToast('Error: ' + (data.message || `HTTP ${res.status}`), 'danger');
                    this.isSaving = false;
                }
            } catch (err) {
                showToast('Error saving records: ' + err.message, 'danger');
                this.isSaving = false;
            }
        },

        // ── Discard all changes ───────────────────────────────────────────────

        async discardChanges() {
            if (this.pendingCount > 0) {
                const ok = await showConfirm('Are you sure you want to discard all changes?', {
                    confirmText: 'Discard', confirmBtnClass: 'btn-warning',
                });
                if (!ok) return;
            }
            location.reload();
        },
    };
}

// Register as an Alpine.data component so the factory is available before Alpine
// processes the DOM even if this script loads after Alpine's defer fires.
document.addEventListener('alpine:init', () => {
    Alpine.data('zoneEditor', zoneEditor);
});
