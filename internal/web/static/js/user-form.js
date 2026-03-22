document.getElementById('toggle-password').addEventListener('click', function() {
    const input = document.getElementById('password');
    const icon = document.getElementById('toggle-password-icon');
    const show = input.type === 'password';
    input.type = show ? 'text' : 'password';
    icon.className = show ? 'bi bi-eye-slash' : 'bi bi-eye';
});
