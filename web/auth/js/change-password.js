async function changePassword() {
    var oldPassword = document.getElementById('old-password').value;
    var newPassword = document.getElementById('new-password').value;
    var confirmPassword = document.getElementById('confirm-password').value;
    var msgEl = document.getElementById('message');
    msgEl.className = 'message';

    if (!oldPassword || !newPassword) {
        msgEl.className = 'message error';
        msgEl.textContent = '请填写所有字段';
        return;
    }
    if (newPassword !== confirmPassword) {
        msgEl.className = 'message error';
        msgEl.textContent = '两次新密码不一致';
        return;
    }

    try {
        var resp = await fetch('/api/change-password', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                old_password: oldPassword,
                new_password: newPassword
            })
        });
        var data = await resp.json();
        if (data.success) {
            msgEl.className = 'message success';
            msgEl.textContent = '密码修改成功';
            document.getElementById('old-password').value = '';
            document.getElementById('new-password').value = '';
            document.getElementById('confirm-password').value = '';
        } else {
            msgEl.className = 'message error';
            msgEl.textContent = data.message || '修改失败';
        }
    } catch (e) {
        msgEl.className = 'message error';
        msgEl.textContent = '网络错误，请重试';
    }
}

document.getElementById('confirm-password').addEventListener('keydown', function(e) {
    if (e.key === 'Enter') changePassword();
});
