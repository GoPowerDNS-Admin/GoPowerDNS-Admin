function zoneTagList() {
    const el = document.getElementById('zone-tags-data');
    const zones = el ? JSON.parse(el.textContent) : [];
    return {
        zones,
        search: '',
        currentPage: 1,
        pageSize: 10,
        get filtered() {
            const q = this.search.trim().toLowerCase();
            if (!q) return this.zones;
            return this.zones.filter(z => z.Name.toLowerCase().includes(q));
        },
        get totalPages() {
            return Math.max(1, Math.ceil(this.filtered.length / this.pageSize));
        },
        get paged() {
            const start = (this.currentPage - 1) * this.pageSize;
            return this.filtered.slice(start, start + this.pageSize);
        },
        init() {
            this.$watch('search', () => { this.currentPage = 1; });
        },
    };
}
