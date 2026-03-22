document.getElementById('copy-secret')?.addEventListener('click', function() {
    const input = document.getElementById('totp-secret');
    navigator.clipboard.writeText(input.value).then(() => {
        this.innerHTML = '<i class="bi bi-clipboard-check"></i>';
        setTimeout(() => { this.innerHTML = '<i class="bi bi-clipboard"></i>'; }, 2000);
    });
});
