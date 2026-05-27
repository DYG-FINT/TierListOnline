async function login() {
    var username = document.getElementById('username').value.trim();
    var password = document.getElementById('password').value;
    var errorEl = document.getElementById('error');

    if (!username || !password) {
        errorEl.textContent = '请填写用户名和密码';
        return;
    }

    try {
        var resp = await fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username: username, password: password })
        });
        var data = await resp.json();
        if (data.success) {
            window.location.href = '/';
        } else {
            errorEl.textContent = data.message || '登录失败';
        }
    } catch (e) {
        errorEl.textContent = '网络错误，请重试';
    }
}

document.getElementById('password').addEventListener('keydown', function(e) {
    if (e.key === 'Enter') login();
});
