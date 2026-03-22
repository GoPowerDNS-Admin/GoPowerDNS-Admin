document.addEventListener('DOMContentLoaded', function() {
    // Zone kind select — show/hide Masters field
    const zoneKindSelect = document.getElementById('zone-kind');
    const mastersField   = document.getElementById('masters-field');
    const mastersInput   = document.getElementById('masters');

    if (zoneKindSelect && mastersField) {
        zoneKindSelect.addEventListener('change', function() {
            if (this.value === 'Slave') {
                mastersField.style.display = 'block';
                mastersInput.required = true;
            } else {
                mastersField.style.display = 'none';
                mastersInput.required = false;
            }
        });
    }

    // Delete zone modal
    const deleteZoneBtn          = document.getElementById('delete-zone-btn');
    const deleteZoneModal        = document.getElementById('deleteZoneModal');
    const deleteZoneConfirmInput = document.getElementById('delete-zone-confirm-input');
    const confirmDeleteBtn       = document.getElementById('confirm-delete-zone-btn');
    const deleteZoneError        = document.getElementById('delete-zone-error');
    const zoneName               = deleteZoneBtn ? deleteZoneBtn.dataset.zoneName : '';

    if (deleteZoneBtn && deleteZoneModal) {
        const modal = new bootstrap.Modal(deleteZoneModal);

        deleteZoneModal.addEventListener('hide.bs.modal', function() {
            if (deleteZoneModal.contains(document.activeElement)) document.activeElement.blur();
        });

        deleteZoneBtn.addEventListener('click', function() {
            deleteZoneConfirmInput.value = '';
            confirmDeleteBtn.disabled = true;
            deleteZoneError.style.display = 'none';
            deleteZoneConfirmInput.classList.remove('is-invalid');
            modal.show();
        });

        deleteZoneConfirmInput.addEventListener('input', function() {
            confirmDeleteBtn.disabled = (this.value !== zoneName);
            if (this.value !== zoneName) return;
            deleteZoneError.style.display = 'none';
            this.classList.remove('is-invalid');
        });

        confirmDeleteBtn.addEventListener('click', function() {
            if (deleteZoneConfirmInput.value !== zoneName) {
                deleteZoneError.style.display = 'block';
                deleteZoneConfirmInput.classList.add('is-invalid');
                return;
            }
            confirmDeleteBtn.disabled = true;
            confirmDeleteBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> Deleting…';

            fetch('/zone/edit/' + encodeURIComponent(zoneName) + '/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
            })
            .then(r => r.json())
            .then(data => {
                if (data.success) {
                    globalThis.location.href = '/dashboard?success=' + encodeURIComponent('Zone deleted successfully');
                } else {
                    alert('Error: ' + data.message);
                    confirmDeleteBtn.disabled = false;
                    confirmDeleteBtn.innerHTML = '<i class="bi bi-trash me-1"></i> Delete Zone';
                }
            })
            .catch(err => {
                alert('Error deleting zone: ' + err.message);
                confirmDeleteBtn.disabled = false;
                confirmDeleteBtn.innerHTML = '<i class="bi bi-trash me-1"></i> Delete Zone';
            });
        });
    }
});
