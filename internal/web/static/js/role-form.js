(function () {
    const checks   = () => document.querySelectorAll('.perm-check');
    const toggles  = () => document.querySelectorAll('.group-toggle');

    // Update a group-toggle tri-state based on its children.
    function syncGroupToggle(group) {
        const kids  = document.querySelectorAll(`.perm-check[data-group="${group}"]`);
        const total = kids.length;
        const checked = Array.from(kids).filter(c => c.checked).length;
        const toggle = document.getElementById('toggle-' + group);
        if (!toggle) return;
        toggle.checked       = checked === total;
        toggle.indeterminate = checked > 0 && checked < total;
    }

    // Sync all group toggles on load.
    const groups = new Set(Array.from(checks()).map(c => c.dataset.group));
    groups.forEach(syncGroupToggle);

    // Group-toggle click → check/uncheck all in group.
    document.addEventListener('change', function (e) {
        if (e.target.classList.contains('group-toggle')) {
            const group = e.target.id.replace('toggle-', '');
            document.querySelectorAll(`.perm-check[data-group="${group}"]`)
                .forEach(c => { c.checked = e.target.checked; });
        }
        if (e.target.classList.contains('perm-check')) {
            syncGroupToggle(e.target.dataset.group);
        }
    });

    // Select-all / Deselect-all buttons.
    document.getElementById('select-all-btn')?.addEventListener('click', function () {
        checks().forEach(c => { c.checked = true; });
        toggles().forEach(t => { t.checked = true; t.indeterminate = false; });
    });
    document.getElementById('deselect-all-btn')?.addEventListener('click', function () {
        checks().forEach(c => { c.checked = false; });
        toggles().forEach(t => { t.checked = false; t.indeterminate = false; });
    });
}());
