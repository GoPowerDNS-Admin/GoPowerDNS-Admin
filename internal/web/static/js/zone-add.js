document.addEventListener('DOMContentLoaded', function () {
    const zoneTypeInput    = document.getElementById('zone-type-input');
    const forwardGroup     = document.getElementById('forward-name-group');
    const reverseGroup     = document.getElementById('reverse-network-group');
    const reverseInput     = document.getElementById('reverse-network');
    const reverseHelp      = document.getElementById('reverse-network-help');
    const computedPreview  = document.getElementById('computed-zone-preview');
    const computedName     = document.getElementById('computed-zone-name');
    const zoneNameInput    = document.getElementById('zone-name');
    const zoneKindSelect   = document.getElementById('zone-kind');
    const mastersGroup     = document.getElementById('masters-group');
    const mastersInput     = document.getElementById('zone-masters');

    // --- Zone category switching ---
    function setCategory(cat) {
        zoneTypeInput.value = cat;

        if (cat === 'forward') {
            forwardGroup.style.display = 'block';
            reverseGroup.style.display = 'none';
            zoneNameInput.required = true;
            reverseInput.required = false;
        } else {
            forwardGroup.style.display = 'none';
            reverseGroup.style.display = 'block';
            zoneNameInput.required = false;
            reverseInput.required = true;
            reverseHelp.textContent = cat === 'reverse-ipv4'
                ? 'Enter an IPv4 network in CIDR notation, e.g. 192.168.1.0/24 or 10.0.0.0/8'
                : 'Enter an IPv6 network in CIDR notation, e.g. 2001:db8::/32 or 2a02:d58:2::/48';
            reverseInput.placeholder = cat === 'reverse-ipv4' ? '192.168.1.0/24' : '2001:db8::/32';
            updatePreview(cat);
        }
    }

    document.getElementById('cat-forward').addEventListener('change', () => setCategory('forward'));
    document.getElementById('cat-reverse-ipv4').addEventListener('change', () => setCategory('reverse-ipv4'));
    document.getElementById('cat-reverse-ipv6').addEventListener('change', () => setCategory('reverse-ipv6'));

    // --- Live reverse zone name preview ---
    reverseInput.addEventListener('input', () => updatePreview(zoneTypeInput.value));

    function updatePreview(cat) {
        const val = reverseInput.value.trim();
        if (!val) { computedPreview.style.display = 'none'; return; }
        const name = cat === 'reverse-ipv4' ? computeReverseIPv4(val) : computeReverseIPv6(val);
        if (name) {
            computedName.textContent = name;
            computedPreview.style.display = 'block';
        } else {
            computedPreview.style.display = 'none';
        }
    }

    function computeReverseIPv4(input) {
        let prefix = 32, ipStr = input;
        if (input.includes('/')) {
            const p = input.split('/');
            ipStr = p[0];
            prefix = parseInt(p[1], 10);
            if (isNaN(prefix) || prefix < 0 || prefix > 32) return null;
        }
        const octs = ipStr.split('.');
        if (octs.length !== 4 || octs.some(o => isNaN(+o) || +o < 0 || +o > 255)) return null;
        const n = Math.max(1, Math.ceil(prefix / 8));
        return octs.slice(0, n).reverse().join('.') + '.in-addr.arpa.';
    }

    function computeReverseIPv6(input) {
        let prefix = 128, ipStr = input;
        if (input.includes('/')) {
            const p = input.split('/');
            ipStr = p[0];
            prefix = parseInt(p[1], 10);
            if (isNaN(prefix) || prefix < 0 || prefix > 128) return null;
        }
        const expanded = expandIPv6(ipStr);
        if (!expanded) return null;
        const n = Math.max(1, Math.ceil(prefix / 4));
        return expanded.slice(0, n).split('').reverse().join('.') + '.ip6.arpa.';
    }

    function expandIPv6(addr) {
        if (addr.includes('::')) {
            const halves = addr.split('::');
            const left   = halves[0] ? halves[0].split(':') : [];
            const right  = halves[1] ? halves[1].split(':') : [];
            const fill   = 8 - left.length - right.length;
            if (fill < 0) return null;
            const groups = [...left, ...Array(fill).fill('0'), ...right];
            return groups.map(g => g.padStart(4, '0')).join('');
        }
        const groups = addr.split(':');
        if (groups.length !== 8) return null;
        return groups.map(g => g.padStart(4, '0')).join('');
    }

    // --- Masters field toggle ---
    function toggleMasters() {
        const isSlave = zoneKindSelect.value === 'Slave';
        mastersGroup.style.display = isSlave ? 'block' : 'none';
        mastersInput.required = isSlave;
    }

    zoneKindSelect.addEventListener('change', toggleMasters);

    // --- Init ---
    setCategory(zoneTypeInput.value || 'forward');
    toggleMasters();
});
