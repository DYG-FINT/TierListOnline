async function register() {
    var username = document.getElementById('username').value.trim();
    var displayName = document.getElementById('display-name').value.trim();
    var password = document.getElementById('password').value;
    var confirmPassword = document.getElementById('confirm-password').value;
    var errorEl = document.getElementById('error');

    if (!username) {
        errorEl.textContent = '请输入用户名';
        return;
    }
    if (!password) {
        errorEl.textContent = '请输入密码';
        return;
    }
    if (password !== confirmPassword) {
        errorEl.textContent = '两次密码不一致';
        return;
    }

    try {
        var resp = await fetch('/api/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                username: username,
                password: password,
                display_name: displayName || undefined
            })
        });
        var data = await resp.json();
        if (data.success) {
            window.location.href = '/';
        } else {
            errorEl.textContent = data.message || '注册失败';
        }
    } catch (e) {
        errorEl.textContent = '网络错误，请重试';
    }
}

document.getElementById('confirm-password').addEventListener('keydown', function(e) {
    if (e.key === 'Enter') register();
});
