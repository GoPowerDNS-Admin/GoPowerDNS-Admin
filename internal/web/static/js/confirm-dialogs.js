// Generic confirmation handler for forms and buttons with data-confirm attribute.
// Replaces inline onsubmit/onclick confirm() patterns.
document.addEventListener('submit', function(e) {
    const msg = e.target.dataset.confirm;
    if (msg && !confirm(msg)) {
        e.preventDefault();
    }
});

document.addEventListener('click', function(e) {
    const btn = e.target.closest('[data-confirm-click]');
    if (btn && !confirm(btn.dataset.confirmClick)) {
        e.preventDefault();
    }
});
