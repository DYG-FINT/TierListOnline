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
    preloadPermissionGroups();
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

var permissionGroupsCache = null;

function renderOnlineUsers() {
    var container = document.getElementById('online-user-list');
    if (!container) return;
    container.innerHTML = '';

    var users = state.onlineUsers || [];
    var canModify = state.canModifyPermissionGroup;

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
        var user = users[i];
        var tr = document.createElement('tr');

        var nameTd = document.createElement('td');
        nameTd.className = 'col-name';
        nameTd.textContent = user.display_name;
        tr.appendChild(nameTd);

        var groupTd = document.createElement('td');
        groupTd.className = 'col-group';

        if (canModify && user.username) {
            var select = document.createElement('select');
            select.className = 'perm-group-select';
            select.setAttribute('data-username', user.username);
            select.setAttribute('data-current-group', user.permission_group);

            populateSelectOptions(select, user.permission_group);

            select.addEventListener('focus', function(e) {
                if (e.target.options.length <= 1) loadPermissionGroupsForSelect(e.target);
            });
            select.addEventListener('mousedown', function(e) {
                if (e.target.options.length <= 1) loadPermissionGroupsForSelect(e.target);
            });
            select.addEventListener('change', function(e) {
                onPermissionGroupChange(e.target);
            });

            groupTd.appendChild(select);
        } else {
            groupTd.textContent = user.permission_group;
        }

        tr.appendChild(groupTd);
        tbody.appendChild(tr);
    }
    table.appendChild(tbody);

    scrollWrap.appendChild(table);
    container.appendChild(scrollWrap);
}

function preloadPermissionGroups() {
    if (!state.canModifyPermissionGroup) return;
    if (permissionGroupsCache) return;

    fetch('/api/permission-groups')
        .then(function(res) { return res.json(); })
        .then(function(data) {
            if (!data.success) {
                showToast(data.message || '无法获取权限组列表', {type: 'error'});
                return;
            }
            permissionGroupsCache = data.groups;
            var selects = document.querySelectorAll('.perm-group-select');
            for (var i = 0; i < selects.length; i++) {
                populateSelectOptions(selects[i], selects[i].getAttribute('data-current-group'));
            }
        })
        .catch(function() {
            showToast('获取权限组列表失败', {type: 'error'});
        });
}

function populateSelectOptions(selectEl, currentValue) {
    if (!permissionGroupsCache) {
        var opt = document.createElement('option');
        opt.value = currentValue;
        opt.textContent = currentValue;
        selectEl.appendChild(opt);
        return;
    }
    selectEl.innerHTML = '';
    for (var i = 0; i < permissionGroupsCache.length; i++) {
        var opt = document.createElement('option');
        opt.value = permissionGroupsCache[i];
        opt.textContent = permissionGroupsCache[i];
        if (permissionGroupsCache[i] === currentValue) {
            opt.selected = true;
        }
        selectEl.appendChild(opt);
    }
}

function loadPermissionGroupsForSelect(selectEl) {
    if (permissionGroupsCache) {
        populateSelectOptions(selectEl, selectEl.getAttribute('data-current-group'));
        return;
    }

    fetch('/api/permission-groups')
        .then(function(res) { return res.json(); })
        .then(function(data) {
            if (!data.success) {
                showToast(data.message || '无法获取权限组列表', {type: 'error'});
                return;
            }
            permissionGroupsCache = data.groups;
            populateSelectOptions(selectEl, selectEl.getAttribute('data-current-group'));
        })
        .catch(function() {
            showToast('获取权限组列表失败', {type: 'error'});
        });
}

function onPermissionGroupChange(selectEl) {
    var newGroup = selectEl.value;
    var username = selectEl.getAttribute('data-username');
    var isSelf = (username === state.username);

    function doModify() {
        fetch('/api/modify-permission-group', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                username: username,
                permission_group: newGroup
            })
        })
        .then(function(res) { return res.json(); })
        .then(function(data) {
            if (data.success) {
                showToast('权限组修改成功', {type: 'success'});
                selectEl.setAttribute('data-current-group', newGroup);
            } else {
                showToast(data.message || '修改失败', {type: 'error'});
                selectEl.value = selectEl.getAttribute('data-current-group');
            }
        })
        .catch(function() {
            showToast('修改权限组失败', {type: 'error'});
            selectEl.value = selectEl.getAttribute('data-current-group');
        });
    }

    if (isSelf) {
        showConfirm('你正在修改自己的权限组，确认继续吗？', function() {
            doModify();
        }, function() {
            selectEl.value = selectEl.getAttribute('data-current-group');
        });
    } else {
        doModify();
    }
}

function showConfirm(message, onConfirm, onCancel) {
    var overlay = document.getElementById('confirm-overlay');
    var msgEl = document.getElementById('confirm-message');
    var cancelBtn = document.getElementById('confirm-cancel-btn');
    var okBtn = document.getElementById('confirm-ok-btn');

    msgEl.textContent = message;
    overlay.classList.add('active');

    function cleanup() {
        overlay.classList.remove('active');
        cancelBtn.removeEventListener('click', onCancelClick);
        okBtn.removeEventListener('click', onOkClick);
        overlay.removeEventListener('click', onOverlayClick);
    }

    function onCancelClick() {
        cleanup();
        if (onCancel) onCancel();
    }

    function onOkClick() {
        cleanup();
        if (onConfirm) onConfirm();
    }

    function onOverlayClick(e) {
        if (e.target === overlay) {
            cleanup();
            if (onCancel) onCancel();
        }
    }

    cancelBtn.addEventListener('click', onCancelClick);
    okBtn.addEventListener('click', onOkClick);
    overlay.addEventListener('click', onOverlayClick);
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
