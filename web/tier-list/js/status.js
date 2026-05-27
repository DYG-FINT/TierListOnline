function toggleStatusOverlay() {
    var overlay = document.getElementById('status-overlay');
    if (overlay.classList.contains('active')) {
        closeStatusOverlay();
    } else {
        openStatusOverlay();
    }
}

function openStatusOverlay() {
    var overlay = document.getElementById('status-overlay');
    overlay.classList.add('active');
    renderStatusOverlay();
}

function closeStatusOverlay() {
    document.getElementById('status-overlay').classList.remove('active');
}

document.getElementById('status-overlay').addEventListener('click', function(e) {
    if (e.target === this) closeStatusOverlay();
});

function renderStatusOverlay() {
    renderOnlineCount();
    renderOnlineUsers();
    renderNameInput();
    renderAuthButtons();
}

function renderOnlineCount() {
    var el = document.getElementById('online-count');
    if (el) {
        el.textContent = '在线: ' + (state.onlineCount || 0) + ' 人';
    }
}

function renderOnlineUsers() {
    var container = document.getElementById('online-user-list');
    if (!container) return;
    container.innerHTML = '';

    var users = state.onlineUsers || [];

    var scrollWrap = document.createElement('div');
    scrollWrap.className = 'online-user-scroll';

    var table = document.createElement('table');
    table.className = 'online-user-table';

    var thead = document.createElement('thead');
    thead.innerHTML =
        '<tr>' +
        '<th class="col-name">名称</th>' +
        '<th class="col-group">权限组</th>' +
        '</tr>';
    table.appendChild(thead);

    var tbody = document.createElement('tbody');
    for (var i = 0; i < users.length; i++) {
        var tr = document.createElement('tr');
        tr.innerHTML =
            '<td class="col-name">' + escapeHTML(users[i].display_name) + '</td>' +
            '<td class="col-group">' + escapeHTML(users[i].permission_group) + '</td>';
        tbody.appendChild(tr);
    }
    table.appendChild(tbody);

    scrollWrap.appendChild(table);
    container.appendChild(scrollWrap);
}

function renderNameInput() {
    var input = document.getElementById('status-name-input');
    if (input && document.activeElement !== input) {
        input.value = state.displayName || '游客';
    }
}

function renderAuthButtons() {
    var container = document.getElementById('status-auth-buttons');
    if (!container) return;
    container.innerHTML = '';

    if (state.isLoggedIn) {
        var changePwdBtn = document.createElement('button');
        changePwdBtn.className = 'btn-auth';
        changePwdBtn.textContent = '修改密码';
        changePwdBtn.onclick = function() {
            window.location.href = '/auth/change-password';
        };
        container.appendChild(changePwdBtn);

        var logoutBtn = document.createElement('button');
        logoutBtn.className = 'btn-danger';
        logoutBtn.textContent = '退出登录';
        logoutBtn.onclick = logout;
        container.appendChild(logoutBtn);
    } else {
        var registerBtn = document.createElement('button');
        registerBtn.className = 'btn-auth';
        registerBtn.textContent = '注册';
        registerBtn.onclick = function() {
            window.location.href = '/auth/register';
        };
        container.appendChild(registerBtn);

        var loginBtn = document.createElement('button');
        loginBtn.className = 'btn-secondary';
        loginBtn.textContent = '登录';
        loginBtn.onclick = function() {
            window.location.href = '/auth/login';
        };
        container.appendChild(loginBtn);
    }
}

function saveDisplayName() {
    var input = document.getElementById('status-name-input');
    var newName = input.value.trim();
    if (!newName) {
        newName = '游客';
        input.value = newName;
    }
    send({type: 'change_display_name', new_display_name: newName});
}

function logout() {
    fetch('/api/logout', {method: 'POST'}).then(function() {
        window.location.reload();
    });
}

// Attach save button handler
(function bindStatusEvents() {
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', bindStatusEvents);
        return;
    }
    var saveBtn = document.getElementById('status-save-name-btn');
    if (saveBtn) {
        saveBtn.addEventListener('click', saveDisplayName);
    }

    var nameInput = document.getElementById('status-name-input');
    if (nameInput) {
        nameInput.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') saveDisplayName();
        });
    }
})();
