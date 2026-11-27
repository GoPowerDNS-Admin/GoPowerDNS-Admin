// Applies the saved or system colour mode before the first paint, so a dark
// page does not flash light while loading. Must be loaded in <head>, before
// any stylesheet. AdminLTE's ColorMode takes over once adminlte.js runs.
(function () {
    const root = document.documentElement;
    if (root.getAttribute('data-lte-color-mode') === 'off') {
        return;
    }

    let stored = null;
    try {
        stored = localStorage.getItem('lte-theme');
    } catch {
        stored = null;
    }

    const authored = root.getAttribute('data-bs-theme');
    let resolved = 'light';
    if (stored === 'dark' || stored === 'light') {
        resolved = stored;
    } else if (authored === 'dark' || authored === 'light') {
        resolved = authored;
    } else if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
        resolved = 'dark';
    }

    root.setAttribute('data-bs-theme', resolved);
    root.style.colorScheme = resolved;
    if (resolved !== authored) {
        root.setAttribute('data-lte-theme-resolved', '');
    }
})();
