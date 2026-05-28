// ========== WebSocket ==========

let ws = null;
let reconnectDelay = 1000;
let reconnectTimer = null;

function connect() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = protocol + '//' + location.host + '/ws';
    ws = new WebSocket(url);

    ws.onopen = function() {
        console.log('WebSocket connected');
        updateStatus(true);
        reconnectDelay = 1000;
    };

    ws.onmessage = function(e) {
        handleMessage(JSON.parse(e.data));
    };

    ws.onclose = function() {
        console.log('WebSocket disconnected');
        updateStatus(false);
        scheduleReconnect();
    };

    ws.onerror = function() {
        ws.close();
    };
}

function scheduleReconnect() {
    clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(function() {
        console.log('Reconnecting...');
        connect();
        reconnectDelay = Math.min(reconnectDelay * 2, 30000);
    }, reconnectDelay);
}

function send(msg) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(msg));
    }
}

function updateStatus(online) {
    var dot = document.getElementById('conn-status');
    if (!dot) return;
    dot.style.background = online ? '#4f4' : '#f44';
    dot.title = online ? '已连接' : '已断开';
}

// ========== Message Handler ==========

function handleMessage(msg) {
    switch (msg.type) {
        case 'full_state':
            state.title = msg.title;
            state.rows = msg.rows;
            state.staging_images = msg.staging_images;
            renderAll();
            break;

        case 'title_updated':
            state.title = msg.title;
            var h1 = document.querySelector('h1');
            if (document.activeElement !== h1) {
                h1.textContent = msg.title;
            }
            break;

        case 'label_updated':
            updateRowInState(msg.row_id, function(r) { r.label = msg.label; });
            updateLabelDOM(msg.row_id, msg.label);
            break;

        case 'label_color_updated':
            updateRowInState(msg.row_id, function(r) { r.color = msg.color; });
            updateLabelColorDOM(msg.row_id, msg.color);
            break;

        case 'row_added':
            state.rows = msg.rows;
            renderRows();
            break;

        case 'row_deleted':
            state.rows = state.rows.filter(function(r) { return r.id !== msg.row_id; });
            removeRowDOM(msg.row_id);
            msg.staging_images.forEach(function(img) {
                state.staging_images.push(img);
                renderCharacter(img, document.getElementById('staging-area'));
            });
            break;

        case 'row_cleared':
            updateRowInState(msg.row_id, function(r) { r.images = []; });
            moveImagesToStaging(msg.row_id);
            msg.staging_images.forEach(function(img) {
                state.staging_images.push(img);
            });
            deselectImage();
            break;

        case 'rows_reordered':
            state.rows = msg.rows;
            renderRows();
            deselectImage();
            break;

        case 'image_uploaded':
            state.staging_images.push(msg.image);
            renderCharacter(msg.image, document.getElementById('staging-area'));
            break;

        case 'image_moved':
            moveImageInState(msg.image_id, msg.target_row_id);
            moveImageDOM(msg.image_id, msg.target_row_id);
            break;

        case 'image_deleted':
            removeImageFromState(msg.image_id);
            removeImageDOM(msg.image_id);
            if (selectedImageId === msg.image_id) deselectImage();
            break;

        case 'all_staged':
            for (var i = 0; i < state.rows.length; i++) {
                state.rows[i].images = [];
            }
            state.staging_images = msg.staging_images;
            moveAllImagesToStaging();
            renderStaging();
            deselectImage();
            break;

        case 'color_sequence_applied':
            state.rows = msg.rows;
            for (var i = 0; i < msg.rows.length; i++) {
                updateLabelColorDOM(msg.rows[i].id, msg.rows[i].color);
            }
            break;

        case 'image_fit_toggled':
            var img = findImageInState(msg.image_id);
            if (img) img.fit_width = msg.fit_width;
            var el = document.querySelector('.character[data-image-id="' + msg.image_id + '"]');
            if (el) {
                if (msg.fit_width) {
                    el.classList.add('fit-width');
                } else {
                    el.classList.remove('fit-width');
                }
            }
            break;

        case 'presets_list':
            updatePresetList(msg.presets);
            break;

        case 'preset_saved':
            if (msg.success) {
                showToast('预设保存成功', {type: 'success'});
                closeSavePresetDialog();
            } else {
                showToast(msg.error || '预设保存失败', {type: 'error'});
            }
            break;

        case 'action_rejected':
            showToast(msg.error || '操作被拒绝', {type: 'error'});
            break;

        case 'upload_rejected':
            showToast(msg.error || '上传被拒绝', {type: 'error'});
            break;

        case 'user_info':
            state.username = msg.username || '';
            state.displayName = msg.display_name || '游客';
            state.permissionGroup = msg.permission_group || 'default';
            state.isLoggedIn = msg.is_logged_in || false;
            state.canModifyPermissionGroup = msg.can_modify_permission_group || false;
            if (!state.isLoggedIn) {
                showToast('当前为游客状态，请点击右上角的状态指示标记进行登录', {type: 'info', duration: 5000});
            }
            if (document.getElementById('status-overlay').classList.contains('active')) {
                renderStatusOverlay();
            }
            break;

        case 'online_users':
            state.onlineCount = msg.count;
            state.onlineUsers = msg.users || [];
            if (state.username) {
                for (var i = 0; i < state.onlineUsers.length; i++) {
                    if (state.onlineUsers[i].username === state.username) {
                        state.permissionGroup = state.onlineUsers[i].permission_group;
                        state.canModifyPermissionGroup = state.onlineUsers[i].can_modify_permission_group || false;
                        break;
                    }
                }
            }
            if (document.getElementById('status-overlay').classList.contains('active')) {
                renderOnlineCount();
                renderOnlineUsers();
            }
            break;
    }
}
