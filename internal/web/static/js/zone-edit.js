/**
 * Zone Edit - DNS Records Management
 * Handles adding, editing, deleting, and saving DNS records for a zone
 */

// DOM selectors
const SELECTORS = {
    RECORDS_TABLE: '#records-table',
    RECORDS_TBODY: '#records-tbody',
    ZONE_DATA: '#zone-data',
    CHANGES_COUNT: '#changes-count',
    CHANGES_INDICATOR: '#changes-indicator',
    ADD_RECORD_BTN: '#add-record-btn',
    SAVE_RECORDS_BTN: '#save-records-btn',
    CANCEL_RECORDS_BTN: '#cancel-records-btn',
    RECORD_MODAL: '#recordModal',
    RECORD_MODAL_LABEL: '#recordModalLabel',
    RECORD_FORM: '#record-form',
    RECORD_ORIGINAL_ID: '#record-original-id',
    RECORD_ORIGINAL_NAME: '#record-original-name',
    RECORD_ORIGINAL_TYPE: '#record-original-type',
    RECORD_ORIGINAL_CONTENT: '#record-original-content',
    RECORD_NAME_INPUT: '#record-name-input',
    RECORD_TYPE_INPUT: '#record-type-input',
    RECORD_TTL_INPUT: '#record-ttl-input',
    RECORD_CONTENT_INPUT: '#record-content-input',
    RECORD_COMMENT_INPUT: '#record-comment-input',
    RECORD_DISABLED_INPUT: '#record-disabled-input',
    SAVE_RECORD_MODAL_BTN: '#save-record-modal-btn',
    TOAST_CONTAINER: '#toast-container'
};

// Add SOA modal selectors
Object.assign(SELECTORS, {
    SOA_MODAL: '#soaModal',
    SOA_MNAME: '#soa-mname',
    SOA_RNAME: '#soa-rname',
    SOA_SERIAL: '#soa-serial',
    SOA_REFRESH: '#soa-refresh',
    SOA_RETRY: '#soa-retry',
    SOA_EXPIRE: '#soa-expire',
    SOA_MINIMUM: '#soa-minimum',
    SAVE_SOA_MODAL_BTN: '#save-soa-modal-btn'
});

/**
 * Show a Bootstrap toast notification
 * @param {string} message - The message to display
 * @param {string} type - The toast type: 'success', 'danger', 'warning', 'info'
 */
function showToast(message, type = 'info') {
    const toastContainer = document.querySelector(SELECTORS.TOAST_CONTAINER);
    if (!toastContainer) {
        console.error('Toast container not found');
        return;
    }

    // Map type to Bootstrap color classes and icons
    const typeConfig = {
        success: { icon: 'bi-check-circle-fill', bg: 'bg-success', title: 'Success' },
        danger: { icon: 'bi-exclamation-triangle-fill', bg: 'bg-danger', title: 'Error' },
        warning: { icon: 'bi-exclamation-circle-fill', bg: 'bg-warning', title: 'Warning' },
        info: { icon: 'bi-info-circle-fill', bg: 'bg-info', title: 'Info' }
    };

    const config = typeConfig[type] || typeConfig.info;
    const toastId = `toast-${Date.now()}`;

    // Create toast element
    const toastHTML = `
        <div id="${toastId}" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
            <div class="toast-header ${config.bg} text-white">
                <i class="bi ${config.icon} me-2"></i>
                <strong class="me-auto">${config.title}</strong>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="toast" aria-label="Close"></button>
            </div>
            <div class="toast-body">
                ${message}
            </div>
        </div>
    `;

    // Add toast to container
    toastContainer.insertAdjacentHTML('beforeend', toastHTML);

    // Initialize and show the toast
    const toastElement = document.getElementById(toastId);
    const toast = new bootstrap.Toast(toastElement, {
        autohide: true,
        delay: 5000
    });

    toast.show();

    // Remove toast element from DOM after it's hidden
    toastElement.addEventListener('hidden.bs.toast', function() {
        toastElement.remove();
    });
}

/**
 * Show a Bootstrap 5 confirm dialog (modal)
 * @param {string} message - Confirmation message
 * @param {{confirmText?: string, cancelText?: string, confirmBtnClass?: string}} [opts]
 * @returns {Promise<boolean>} resolves true if confirmed, false otherwise
 */
function showConfirm(message, opts = {}) {
    const options = Object.assign({
        confirmText: 'Confirm',
        cancelText: 'Cancel',
        confirmBtnClass: 'btn-danger'
    }, opts);

    // Create reusable modal once
    let modalEl = document.getElementById('genericConfirmModal');
    if (!modalEl) {
        const html = `
            <div class="modal fade" id="genericConfirmModal" tabindex="-1" aria-hidden="true">
              <div class="modal-dialog modal-dialog-centered">
                <div class="modal-content">
                  <div class="modal-header">
                    <h5 class="modal-title">Please Confirm</h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                  </div>
                  <div class="modal-body">
                    <p id="genericConfirmMessage" class="mb-0"></p>
                  </div>
                  <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" data-bs-dismiss="modal" id="genericConfirmCancelBtn">Cancel</button>
                    <button type="button" class="btn btn-danger" id="genericConfirmOkBtn">Confirm</button>
                  </div>
                </div>
              </div>
            </div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        modalEl = document.getElementById('genericConfirmModal');
    }

    // Update content and buttons
    modalEl.querySelector('#genericConfirmMessage').textContent = message;
    const cancelBtn = modalEl.querySelector('#genericConfirmCancelBtn');
    const okBtn = modalEl.querySelector('#genericConfirmOkBtn');
    cancelBtn.textContent = options.cancelText;
    okBtn.textContent = options.confirmText;
    // reset classes
    okBtn.className = 'btn ' + options.confirmBtnClass;

    // Return promise that resolves on action
    return new Promise((resolve) => {
        const bsModal = bootstrap.Modal.getOrCreateInstance(modalEl);

        const handleOk = () => {
            cleanup();
            resolve(true);
            bsModal.hide();
        };
        const handleCancel = () => {
            cleanup();
            resolve(false);
        };
        const handleHidden = () => {
            cleanup();
            resolve(false);
        };

        function cleanup() {
            okBtn.removeEventListener('click', handleOk);
            cancelBtn.removeEventListener('click', handleCancel);
            modalEl.removeEventListener('hidden.bs.modal', handleHidden);
        }

        okBtn.addEventListener('click', handleOk, { once: true });
        cancelBtn.addEventListener('click', handleCancel, { once: true });
        modalEl.addEventListener('hidden.bs.modal', handleHidden, { once: true });

        bsModal.show();
    });
}

// Generate record ID
function generateRecordId(name, type, content) {
    return `${name}-${type}-${content}`;
}

/**
 * Canonicalize a hostname (for record content, not name)
 * Only appends zone name if it's a relative subdomain
 * @param {string} hostname - The hostname to canonicalize
 * @returns {string} The canonical hostname with trailing dot
 */
function canonicalizeHostname(hostname) {
    if (hostname === null || hostname === undefined || hostname === '') {
        return hostname;
    }
    hostname = hostname.trim();

    // If already ends with a dot, it's canonical
    if (hostname.endsWith('.')) {
        return hostname;
    }

    // Check if it contains a dot - if so, it's likely a FQDN to another domain
    // Just add the trailing dot
    if (hostname.includes('.')) {
        return hostname + '.';
    }

    // No dots mean it's a simple name - could be relative to this zone
    // Check if user wants zone-relative (e.g., "server") or external (e.g., "localhost")
    // For safety, if it's a single label, just add a dot (treat as FQDN)
    return hostname + '.';
}

// Parse SOA content string into fields
function parseSOA(content) {
    if (!content) return null;
    const parts = content.trim().split(/\s+/);
    if (parts.length < 7) return null;
    return {
        mname: parts[0],
        rname: parts[1],
        serial: parts[2],
        refresh: parts[3],
        retry: parts[4],
        expire: parts[5],
        minimum: parts[6]
    };
}

// Compose SOA content string from fields
function composeSOA(fields) {
    const intFields = ['serial','refresh','retry','expire','minimum'];
    for (const k of intFields) {
        if (fields[k] === '' || fields[k] === null || fields[k] === undefined) return null;
        const n = Number.parseInt(String(fields[k]), 10);
        if (!Number.isFinite(n) || n < 0) return null;
        fields[k] = String(n);
    }
    let mname = canonicalizeHostname(fields.mname || '');
    let rname = canonicalizeHostname(fields.rname || '');
    if (!mname || !rname) return null;
    return `${mname} ${rname} ${fields.serial} ${fields.refresh} ${fields.retry} ${fields.expire} ${fields.minimum}`;
}

/**
 * Canonicalize content for record types that require FQDN
 * @param {string} type - The DNS record type
 * @param {string} content - The record content to canonicalize
 * @returns {string} The canonical content
 */
function canonicalizeContent(type, content) {
    if (content === null || content === undefined || content === '') {
        return content;
    }
    content = content.trim();

    // Record types that require canonical names in content
    const typesRequiringCanonical = ['CNAME', 'MX', 'NS', 'PTR', 'SRV'];

    if (typesRequiringCanonical.includes(type)) {
        // For MX and SRV records, only canonicalize the hostname part
        if (type === 'MX') {
            // MX format: "priority hostname"
            const parts = content.split(/\s+/);
            if (parts.length >= 2) {
                const priority = parts[0];
                const hostname = canonicalizeHostname(parts.slice(1).join(' '));
                return `${priority} ${hostname}`;
            }
        } else if (type === 'SRV') {
            // SRV format: "priority weight port target"
            const parts = content.split(/\s+/);
            if (parts.length >= 4) {
                const priority = parts[0];
                const weight = parts[1];
                const port = parts[2];
                const target = canonicalizeHostname(parts.slice(3).join(' '));
                return `${priority} ${weight} ${port} ${target}`;
            }
        } else {
            // CNAME, NS, PTR - entire content should be canonical
            return canonicalizeHostname(content);
        }
    }

    return content;
}

$(document).ready(function() {
    // Only run if we're on the zone edit page
    const hasRecordsTable = $(SELECTORS.RECORDS_TABLE).length > 0;
    if (hasRecordsTable === false) {
        return;
    }

    // Get zone name from data attribute
    const zoneName = $(SELECTORS.ZONE_DATA).data('zone-name');

    // Track which records existed in the database on page load
    const originalRecords = new Set();
    $(SELECTORS.RECORDS_TABLE).find('tbody tr').each(function() {
        const fullName = $(this).data('full-name');
        const type = $(this).find('td').eq(1).text();
        if (fullName && type) {
            originalRecords.add(`${fullName}|${type}`);
        }
    });

    // Track changes to records
    const pendingChanges = new Map();

    // Initialize DataTable
    let dataTable = null;
    if ($.fn.DataTable) {
        dataTable = $(SELECTORS.RECORDS_TABLE).DataTable({
            order: [[0, 'asc']], // Sort by Name column by default
            pageLength: 25,
            lengthMenu: [[10, 25, 50, 100, -1], [10, 25, 50, 100, "All"]],
            columnDefs: [
                {
                    targets: -1, // Last column (Actions)
                    orderable: false,
                    searchable: false
                }
            ],
            language: {
                search: "Search records:",
                lengthMenu: "Show _MENU_ records per page",
                info: "Showing _START_ to _END_ of _TOTAL_ records",
                infoEmpty: "No records available",
                infoFiltered: "(filtered from _TOTAL_ total records)",
                zeroRecords: "No matching records found"
            }
        });
    }

    // Get display name (strip zone name for display)
    function getDisplayName(fullName) {
        if (fullName === null || fullName === undefined || fullName === '') {
            return fullName;
        }
        // If it's the zone itself, return @
        if (fullName === zoneName || fullName === zoneName.replace(/\.$/, '')) {
            return '@';
        }
        // Strip the zone name suffix for display
        const zoneWithoutDot = zoneName.replace(/\.$/, '');
        if (fullName.endsWith('.' + zoneWithoutDot + '.')) {
            return fullName.replace('.' + zoneWithoutDot + '.', '');
        } else if (fullName.endsWith('.' + zoneWithoutDot)) {
            return fullName.replace('.' + zoneWithoutDot, '');
        }
        return fullName;
    }


    // Update changes indicator
    function updateChangesIndicator() {
        const count = pendingChanges.size;
        if (count > 0) {
            $(SELECTORS.CHANGES_COUNT).text(count);
            $(SELECTORS.CHANGES_INDICATOR).show();
        } else {
            $(SELECTORS.CHANGES_INDICATOR).hide();
        }
    }

    // Ensure DNS name is in canonical format (ends with a dot)
    // Automatically appends zone name if only subdomain is provided
    function canonicalizeName(name) {
        if (name === null || name === undefined || name === '') {
            return name;
        }
        name = name.trim();

        // Handle @ as zone apex
        if (name === '@') {
            return zoneName;
        }

        // If the name doesn't end with a dot and doesn't already include the zone name,
        // it's just a subdomain - append the zone name
        const endsWithDot = name.endsWith('.');
        if (endsWithDot === false) {
            // Check if this is already a fully qualified name that includes the zone
            const endsWithZoneName = name.endsWith(zoneName.replace(/\.$/, ''));
            if (endsWithZoneName) {
                // It's already fully qualified, just add the trailing dot
                name += '.';
            } else {
                // It's just a subdomain, append zone name
                name = name + '.' + zoneName;
            }
        }

        return name;
    }

    // Add or update record in UI
    function updateRecordRow(id, record, isNew = false) {
        const displayName = getDisplayName(record.name);
        const comment = record.comment || '';

        // Row data for DataTable
        const rowData = [
            displayName,
            record.type,
            record.ttl,
            `<span class="badge ${record.disabled ? 'bg-danger' : 'bg-success'}">${record.disabled ? 'Disabled' : 'Active'}</span>`,
            `<span class="text-truncate" style="max-width: 200px; display: inline-block;" title="${record.content}">${record.content}</span>`,
            `<span class="text-truncate" style="max-width: 150px; display: inline-block;" title="${comment}">${comment}</span>`,
            `<button type="button" class="btn btn-sm btn-primary edit-record-btn" title="Edit"><i class="bi bi-pencil"></i></button> <button type="button" class="btn btn-sm btn-danger delete-record-btn" title="Delete"><i class="bi bi-trash"></i></button>`
        ];

        if (dataTable) {
            // Find existing row
            let foundRow = null;
            dataTable.rows().every(function(rowIdx) {
                const node = dataTable.row(rowIdx).node();
                if ($(node).data('record-id') === id) {
                    foundRow = dataTable.row(rowIdx);
                    return false;
                }
            });

            if (foundRow) {
                // Update existing row
                foundRow.data(rowData);
                const $node = $(foundRow.node());
                $node.attr('data-full-name', record.name);
                $node.attr('data-comment', comment);
                $node.attr('data-record-id', id);
                if (isNew === false) {
                    $node.addClass('table-warning');
                }
            } else {
                // Add new row
                const newRow = dataTable.row.add(rowData).draw(false);
                const $node = $(newRow.node());
                $node.attr('data-record-id', id);
                $node.attr('data-full-name', record.name);
                $node.attr('data-comment', comment);
                if (isNew) {
                    $node.addClass('table-success');
                }
            }
        } else {
            // Fallback to non-DataTable method
            let $row = $('tr').filter(function() {
                return $(this).data('record-id') === id;
            });
            if ($row.length === 0) {
                // Create row element safely to avoid issues with special characters in attributes
                $row = $('<tr>').addClass(isNew ? 'table-success' : '');
                $row.attr('data-record-id', id);
                $row.attr('data-full-name', record.name);
                $row.attr('data-comment', comment);
                $row.append($('<td>').addClass('record-name').text(displayName));
                $row.append($('<td>').addClass('record-type').text(record.type));
                $row.append($('<td>').addClass('record-ttl').text(record.ttl));
                $row.append($('<td>').addClass('record-status').html(rowData[3]));
                $row.append($('<td>').addClass('record-content').html(rowData[4]));
                $row.append($('<td>').addClass('record-comment').html(rowData[5]));
                $row.append($('<td>').html(rowData[6]));
                $(SELECTORS.RECORDS_TBODY).append($row);
            } else {
                $row.attr('data-full-name', record.name);
                $row.attr('data-comment', comment);
                $row.find('.record-name').text(displayName);
                $row.find('.record-type').text(record.type);
                $row.find('.record-ttl').text(record.ttl);
                $row.find('.record-status').html(rowData[3]);
                $row.find('.record-content').html(rowData[4]);
                $row.find('.record-comment').html(rowData[5]);
                if (isNew === false) {
                    $row.addClass('table-warning');
                }
            }
        }
    }

    // Get modal instance
    const modalElement = document.querySelector(SELECTORS.RECORD_MODAL);
    const modal = new bootstrap.Modal(modalElement);
    const soaModalElement = document.querySelector(SELECTORS.SOA_MODAL);
    const soaModal = soaModalElement ? new bootstrap.Modal(soaModalElement) : null;

    // Fix aria-hidden accessibility issue: remove focus before modal is hidden
    modalElement.addEventListener('hide.bs.modal', function() {
        // Remove focus from any element inside the modal
        const focusedElement = document.activeElement;
        if (focusedElement && modalElement.contains(focusedElement)) {
            focusedElement.blur();
        }
    });

    // Helper: update help text for the Data field based on selected record type
    function updateRecordContentHelp() {
        const helpEl = document.getElementById('record-content-help');
        if (!helpEl) return;
        const $typeSel = $(SELECTORS.RECORD_TYPE_INPUT);
        const $opt = $typeSel.find('option:selected');
        const help = ($opt.data('help') || '').toString();
        if (help && help.trim().length > 0) {
            helpEl.textContent = help;
        } else {
            helpEl.textContent = 'Record data (e.g., IP address, hostname, text). For CNAME, MX, NS, PTR, and SRV records, enter the full domain name (e.g., cdn.cloudflare.com). A trailing dot will be added automatically.';
        }
    }

    // React on record type change in modal
    $(document).on('change', SELECTORS.RECORD_TYPE_INPUT, function() {
        updateRecordContentHelp();
    });

    // Open modal for adding new record
    $(SELECTORS.ADD_RECORD_BTN).on('click', function() {
        $(SELECTORS.RECORD_MODAL_LABEL).text('Add Record');
        $(SELECTORS.RECORD_FORM)[0].reset();
        $(SELECTORS.RECORD_ORIGINAL_ID).val('');
        $(SELECTORS.RECORD_ORIGINAL_NAME).val('');
        $(SELECTORS.RECORD_ORIGINAL_TYPE).val('');
        $(SELECTORS.RECORD_ORIGINAL_CONTENT).val('');
        $(SELECTORS.RECORD_TTL_INPUT).val('3600');
        $(SELECTORS.RECORD_COMMENT_INPUT).val('');
        $(SELECTORS.SAVE_RECORD_MODAL_BTN).html('<i class="bi bi-plus-circle"></i> Add Record');
        // Initialize help text for default-selected type
        updateRecordContentHelp();
        modal.show();
    });

    // Open modal for editing record
    $(document).on('click', '.edit-record-btn', function() {
        const $row = $(this).closest('tr');
        const id = $row.data('record-id');
        const fullName = $row.data('full-name'); // Full canonical name
        const displayName = $row.find('td').eq(0).text(); // Name column
        const type = $row.find('td').eq(1).text(); // Type column
        const ttl = $row.find('td').eq(2).text(); // TTL column
        const content = $row.find('td').eq(4).find('span').attr('title') || $row.find('td').eq(4).text(); // Data column
        const disabled = $row.find('td').eq(3).find('.badge').hasClass('bg-danger'); // Status column
        const comment = $row.data('comment') || '';

        // SOA special editor
        if (type === 'SOA' && soaModal) {
            const soa = parseSOA(content);
            if (!soa) {
                showToast('Malformed SOA content; cannot parse existing values.', 'danger');
            }
            $(SELECTORS.RECORD_ORIGINAL_ID).val(id);
            $(SELECTORS.RECORD_ORIGINAL_NAME).val(fullName);
            $(SELECTORS.RECORD_ORIGINAL_TYPE).val(type);
            $(SELECTORS.RECORD_ORIGINAL_CONTENT).val(content);

            // Fill fields if parsed
            if (soa) {
                $(SELECTORS.SOA_MNAME).val(soa.mname);
                $(SELECTORS.SOA_RNAME).val(soa.rname);
                $(SELECTORS.SOA_SERIAL).val(soa.serial);
                $(SELECTORS.SOA_REFRESH).val(soa.refresh);
                $(SELECTORS.SOA_RETRY).val(soa.retry);
                $(SELECTORS.SOA_EXPIRE).val(soa.expire);
                $(SELECTORS.SOA_MINIMUM).val(soa.minimum);
            }
            // Store auxiliary info for saving
            $(SELECTORS.SOA_MODAL).data('origTtl', ttl);
            $(SELECTORS.SOA_MODAL).data('origDisabled', disabled);
            $(SELECTORS.SOA_MODAL).data('origComment', comment);
            $(SELECTORS.SOA_MODAL).data('displayName', displayName);
            if (soaModal) soaModal.show();
            return; // do not open generic modal
        }

        $(SELECTORS.RECORD_MODAL_LABEL).text('Edit Record');
        $(SELECTORS.RECORD_ORIGINAL_ID).val(id);
        $(SELECTORS.RECORD_ORIGINAL_NAME).val(fullName);
        $(SELECTORS.RECORD_ORIGINAL_TYPE).val(type);
        $(SELECTORS.RECORD_ORIGINAL_CONTENT).val(content);
        $(SELECTORS.RECORD_NAME_INPUT).val(displayName);
        $(SELECTORS.RECORD_TYPE_INPUT).val(type);
        $(SELECTORS.RECORD_TTL_INPUT).val(ttl);
        $(SELECTORS.RECORD_CONTENT_INPUT).val(content);
        $(SELECTORS.RECORD_COMMENT_INPUT).val(comment);
        $(SELECTORS.RECORD_DISABLED_INPUT).prop('checked', disabled);
        $(SELECTORS.SAVE_RECORD_MODAL_BTN).html('<i class="bi bi-pencil"></i> Update Record');
        // Initialize help text for the selected record type
        updateRecordContentHelp();
        modal.show();
    });

    // Save SOA from modal
    $(SELECTORS.SAVE_SOA_MODAL_BTN).on('click', function() {
        if (!soaModal) return;
        const originalId = $(SELECTORS.RECORD_ORIGINAL_ID).val();
        const originalName = $(SELECTORS.RECORD_ORIGINAL_NAME).val();
        const originalType = $(SELECTORS.RECORD_ORIGINAL_TYPE).val();
        const originalContent = $(SELECTORS.RECORD_ORIGINAL_CONTENT).val();
        if (originalType !== 'SOA') { soaModal.hide(); return; }

        const fields = {
            mname: $(SELECTORS.SOA_MNAME).val().trim(),
            rname: $(SELECTORS.SOA_RNAME).val().trim(),
            serial: $(SELECTORS.SOA_SERIAL).val().trim(),
            refresh: $(SELECTORS.SOA_REFRESH).val().trim(),
            retry: $(SELECTORS.SOA_RETRY).val().trim(),
            expire: $(SELECTORS.SOA_EXPIRE).val().trim(),
            minimum: $(SELECTORS.SOA_MINIMUM).val().trim()
        };
        const composed = composeSOA(fields);
        if (!composed) {
            showToast('Please provide valid SOA values.', 'danger');
            return;
        }

        const record = {
            name: originalName,
            type: 'SOA',
            ttl: Number.parseInt($(SELECTORS.SOA_MODAL).data('origTtl')) || 0,
            content: composed,
            disabled: !!$(SELECTORS.SOA_MODAL).data('origDisabled'),
            comment: ($(SELECTORS.SOA_MODAL).data('origComment') || '').toString()
        };

        const newId = generateRecordId(record.name, record.type, record.content);
        let isChanged = (originalContent !== record.content);
        if (!isChanged) {
            showToast('No changes detected for this SOA record.', 'info');
            soaModal.hide();
            return;
        }

        const key = `${record.name}|${record.type}`;
        const hasExistingChange = pendingChanges.has(key);
        if (!hasExistingChange) {
            const existedInDB = originalRecords.has(key);
            pendingChanges.set(key, {
                name: record.name,
                type: record.type,
                ttl: record.ttl,
                comment: record.comment,
                records: [],
                existed: existedInDB,
                changed: true
            });
        }
        const change = pendingChanges.get(key);
        change.changed = true;
        change.ttl = record.ttl;
        change.comment = record.comment;
        // Ensure single SOA record: remove all existing and set one
        change.records = [{ content: record.content, disabled: record.disabled }];

        // Update UI
        updateRecordRow(newId, record, false);
        // Remove old row if id changed (name/type same typically, content changes)
        if (originalId && originalId !== newId) {
            $('tr').filter(function() { return $(this).data('record-id') === originalId; }).remove();
        }
        updateChangesIndicator();
        soaModal.hide();
    });

    // Save record from modal
    $(SELECTORS.SAVE_RECORD_MODAL_BTN).on('click', function() {
        const form = $(SELECTORS.RECORD_FORM)[0];
        const isValid = form.checkValidity();
        if (isValid === false) {
            form.reportValidity();
            return;
        }

        const originalId = $(SELECTORS.RECORD_ORIGINAL_ID).val();
        const originalName = $(SELECTORS.RECORD_ORIGINAL_NAME).val();
        const originalType = $(SELECTORS.RECORD_ORIGINAL_TYPE).val();
        const originalContent = $(SELECTORS.RECORD_ORIGINAL_CONTENT).val();
        const recordType = $(SELECTORS.RECORD_TYPE_INPUT).val();
        const record = {
            name: canonicalizeName($(SELECTORS.RECORD_NAME_INPUT).val()),
            type: recordType,
            ttl: Number.parseInt($(SELECTORS.RECORD_TTL_INPUT).val()),
            content: canonicalizeContent(recordType, $(SELECTORS.RECORD_CONTENT_INPUT).val()),
            disabled: $(SELECTORS.RECORD_DISABLED_INPUT).is(':checked'),
            comment: $(SELECTORS.RECORD_COMMENT_INPUT).val().trim()
        };

        const newId = generateRecordId(record.name, record.type, record.content);
        const isNewRecord = (originalId === '' || originalId === null || originalId === undefined);

        // Determine if anything actually changed
        let isChanged = isNewRecord;
        if (!isNewRecord) {
            const $origRow = $('tr').filter(function() {
                return $(this).data('record-id') === originalId;
            });
            if ($origRow.length > 0) {
                const rowTTL = Number.parseInt($origRow.find('td').eq(2).text()) || 0;
                const rowDisabled = $origRow.find('td').eq(3).find('.badge').hasClass('bg-danger');
                const rowComment = $origRow.data('comment') || '';
                // Name and Type we have in hidden fields
                isChanged = (
                    originalName !== record.name ||
                    originalType !== record.type ||
                    originalContent !== record.content ||
                    rowTTL !== record.ttl ||
                    rowDisabled !== record.disabled ||
                    rowComment !== record.comment
                );
            } else {
                // If row not found, treat as changed to be safe
                isChanged = true;
            }
        }

        if (!isChanged) {
            showToast('No changes detected for this record.', 'info');
            modal.hide();
            return;
        }

        // If editing and name/type changed, remove old record from old key
        if (!isNewRecord && (originalName !== record.name || originalType !== record.type)) {
            const oldKey = `${originalName}|${originalType}`;

            // Check if the old record existed in the database on page load
            if (originalRecords.has(oldKey)) {
                // Record existed in database, mark for deletion with empty record set
                if (pendingChanges.has(oldKey)) {
                    // Already have changes for this key, filter out this content
                    const oldChange = pendingChanges.get(oldKey);
                    oldChange.records = oldChange.records.filter(r => r.content !== originalContent);
                } else {
                    // Create empty record set to mark for deletion
                    pendingChanges.set(oldKey, {
                        name: originalName,
                        type: originalType,
                        ttl: 0,
                        comment: '',
                        records: [],
                        existed: true // This RRset existed in the database
                    });
                }
            } else {
                // Record was added in this session, just remove the content
                if (pendingChanges.has(oldKey)) {
                    const oldChange = pendingChanges.get(oldKey);
                    oldChange.records = oldChange.records.filter(r => r.content !== originalContent);
                    // If no records left, remove the key entirely (never existed in DB)
                    if (oldChange.records.length === 0) {
                        pendingChanges.delete(oldKey);
                    }
                }
            }
        }

        // Add to pending changes with new key
        const key = `${record.name}|${record.type}`;
        const hasExistingChange = pendingChanges.has(key);
        if (hasExistingChange === false) {
            // Check if this is a new RRset or existed in database
            const existedInDB = originalRecords.has(key);
            pendingChanges.set(key, {
                name: record.name,
                type: record.type,
                ttl: record.ttl,
                comment: record.comment,
                records: [],
                existed: existedInDB, // Flag indicating if this RRset existed in the database
                changed: true
            });
        }

        const change = pendingChanges.get(key);
        change.changed = true;
        change.ttl = record.ttl;
        change.comment = record.comment;

        // If editing and only content changed (same name/type), remove old content
        if (!isNewRecord && originalName === record.name && originalType === record.type && originalContent !== record.content) {
            change.records = change.records.filter(r => r.content !== originalContent);
        }

        // Check if this content already exists for this name/type
        const existingIndex = change.records.findIndex(r => r.content === record.content);
        if (existingIndex >= 0) {
            // Update existing record
            change.records[existingIndex] = {
                content: record.content,
                disabled: record.disabled
            };
        } else {
            // Add new record
            change.records.push({
                content: record.content,
                disabled: record.disabled
            });
        }

        // Update UI
        updateRecordRow(newId, record, isNewRecord);

        // Remove old row if name/type/content changed
        if (originalId && originalId !== newId) {
            $('tr').filter(function() {
                return $(this).data('record-id') === originalId;
            }).remove();
        }

        updateChangesIndicator();
        modal.hide();
    });

    // Delete record
    $(document).on('click', '.delete-record-btn', function() {
        const $btn = $(this);
        showConfirm('Are you sure you want to delete this record?', {
            confirmText: 'Delete',
            confirmBtnClass: 'btn-danger'
        }).then(function(confirmed) {
            if (!confirmed) {
                return;
            }

            const $row = $btn.closest('tr');
            const fullName = $row.data('full-name') || $row.find('.record-name').text();
            const type = $row.find('td').eq(1).text();
            const content = $row.find('td').eq(4).find('span').attr('title') || $row.find('td').eq(4).text();

            // Mark for deletion by removing from pending changes or adding empty record set
            const key = `${fullName}|${type}`;

            // Check if the record existed in the database on page load
            if (originalRecords.has(key)) {
                // Record existed in database, mark for deletion
                if (pendingChanges.has(key)) {
                    // Already have changes for this key, filter out this content
                    const change = pendingChanges.get(key);
                    change.records = change.records.filter(r => r.content !== content);
                    change.changed = true;
                } else {
                    // Create empty record set to mark for deletion
                    pendingChanges.set(key, {
                        name: fullName,
                        type: type,
                        ttl: 0,
                        comment: '',
                        records: [],
                        existed: true, // This RRset existed in the database
                        changed: true
                    });
                }
            } else {
                // Record was added in this session, just remove it from pending changes
                if (pendingChanges.has(key)) {
                    const change = pendingChanges.get(key);
                    change.records = change.records.filter(r => r.content !== content);
                    // If no records left, remove the key entirely (never existed in DB)
                    if (change.records.length === 0) {
                        pendingChanges.delete(key);
                    }
                }
            }

            // Remove row using DataTable API if available
            if (dataTable) {
                dataTable.row($row).remove().draw(false);
            } else {
                $row.remove();
            }

            updateChangesIndicator();
        });
    });

    // Save all changes
    $(SELECTORS.SAVE_RECORDS_BTN).on('click', function() {
        if (pendingChanges.size === 0) {
            showToast('No changes to save', 'warning');
            return;
        }

        const changes = Array.from(pendingChanges.values());

        $(this).prop('disabled', true).html('<span class="spinner-border spinner-border-sm"></span> Saving...');

        $.ajax({
            url: `/zone/edit/${zoneName}/records`,
            method: 'POST',
            contentType: 'application/json',
            data: JSON.stringify({ changes: changes }),
            success: function(response) {
                if (response.success) {
                    showToast('Records saved successfully!', 'success');
                    setTimeout(() => {
                        globalThis.location.reload();
                    }, 1000);
                } else {
                    showToast('Error: ' + response.message, 'danger');
                    $(SELECTORS.SAVE_RECORDS_BTN).prop('disabled', false).html('<i class="bi bi-save"></i> Save Changes');
                }
            },
            error: function(xhr) {
                const response = xhr.responseJSON || {};
                showToast('Error saving records: ' + (response.message || 'Unknown error'), 'danger');
                $(SELECTORS.SAVE_RECORDS_BTN).prop('disabled', false).html('<i class="bi bi-save"></i> Save Changes');
            }
        });
    });

    // Cancel changes
    $(SELECTORS.CANCEL_RECORDS_BTN).on('click', function() {
        if (pendingChanges.size > 0) {
            showConfirm('Are you sure you want to discard all changes?', {
                confirmText: 'Discard',
                confirmBtnClass: 'btn-warning'
            }).then(function(confirmed) {
                if (confirmed) {
                    globalThis.location.reload();
                }
            });
            return;
        }
        globalThis.location.reload();
    });
});
