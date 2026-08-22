(function () {
    const source = document.getElementById('source');

    // External groups render the source as a locked (disabled) field and their
    // members read-only, so there is nothing to toggle client-side.
    if (!source || source.disabled) {
        return;
    }

    const box       = document.getElementById('group-members-box');
    const help      = document.getElementById('group-members-help');
    const note      = document.getElementById('group-members-external-note');
    const extIdHelp = document.getElementById('external-id-help');

    const LOCAL_MEMBERS_HELP = 'Select users who should be members of this group';
    const EXT_MEMBERS_HELP   = 'Membership is managed by the selected directory and synced on login.';
    const LOCAL_EXTID_HELP   = 'Optional. Required when source is external.';
    const EXT_EXTID_HELP      = 'Required — the directory identifier (DN for LDAP, claim for OIDC).';

    function isExternal() {
        return source.value === 'ldap' || source.value === 'oidc';
    }

    function apply() {
        const ext = isExternal();

        if (box) {
            box.classList.toggle('opacity-50', ext);
            // Disabled checkboxes are not submitted, so an external selection cannot
            // send member data the server would silently discard.
            box.querySelectorAll('input[name="user_ids"]').forEach(function (cb) {
                cb.disabled = ext;
            });
        }

        if (help) {
            help.textContent = ext ? EXT_MEMBERS_HELP : LOCAL_MEMBERS_HELP;
        }

        if (note) {
            note.classList.toggle('d-none', !ext);
        }

        if (extIdHelp) {
            extIdHelp.textContent = ext ? EXT_EXTID_HELP : LOCAL_EXTID_HELP;
        }
    }

    source.addEventListener('change', apply);

    // Reflect the initial source (e.g. after a validation-failed re-render).
    apply();
}());
