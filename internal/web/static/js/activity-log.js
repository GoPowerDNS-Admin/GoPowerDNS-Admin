document.addEventListener('DOMContentLoaded', function () {
    function onChange(ev) {
        var u = new URL(window.location.href);
        u.searchParams.set('pageSize', ev.target.value);
        u.searchParams.set('page', '1');
        window.location.assign(u.toString());
    }

    ['activity-page-size', 'activity-page-size-bottom'].forEach(function (id) {
        var el = document.getElementById(id);
        if (el) el.addEventListener('change', onChange);
    });
});
