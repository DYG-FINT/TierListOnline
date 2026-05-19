// ========== Client State ==========

const state = {
    title: '',
    bg_color: '#1a1a1a',
    rows: [],
    staging_images: []
};

let activeRowId = null;
let dragImageId = null;
let ws = null;
let reconnectDelay = 1000;
let reconnectTimer = null;
let labelDebounceTimer = null;
let titleDebounceTimer = null;

// ========== WebSocket ==========

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
            break;

        case 'rows_reordered':
            state.rows = msg.rows;
            renderRows();
            break;

        case 'image_uploaded':
            state.staging_images.push(msg.image);
            renderCharacter(msg.image, document.getElementById('staging-area'));
            break;

        case 'image_moved':
            moveImageInState(msg.image_id, msg.target_row_id, msg.target_index);
            moveImageDOM(msg.image_id, msg.target_row_id, msg.target_index);
            break;

        case 'image_deleted':
            removeImageFromState(msg.image_id);
            removeImageDOM(msg.image_id);
            break;

        case 'all_staged':
            // Move all images from rows to staging
            for (var i = 0; i < state.rows.length; i++) {
                state.rows[i].images = [];
            }
            state.staging_images = msg.staging_images;
            moveAllImagesToStaging();
            renderStaging();
            break;

        case 'color_sequence_applied':
            state.rows = msg.rows;
            // Update label holder colors in DOM without rebuilding
            for (var i = 0; i < msg.rows.length; i++) {
                updateLabelColorDOM(msg.rows[i].id, msg.rows[i].color);
            }
            break;
    }
}

// ========== State Helpers ==========

function updateRowInState(rowId, fn) {
    for (var i = 0; i < state.rows.length; i++) {
        if (state.rows[i].id === rowId) {
            fn(state.rows[i]);
            break;
        }
    }
}

function findImageInState(imageId) {
    for (var i = 0; i < state.rows.length; i++) {
        for (var j = 0; j < state.rows[i].images.length; j++) {
            if (state.rows[i].images[j].id === imageId) return state.rows[i].images[j];
        }
    }
    for (var k = 0; k < state.staging_images.length; k++) {
        if (state.staging_images[k].id === imageId) return state.staging_images[k];
    }
    return null;
}

function moveImageInState(imageId, targetRowId, position) {
    var img = null;
    // Remove from source
    for (var i = 0; i < state.rows.length; i++) {
        for (var j = 0; j < state.rows[i].images.length; j++) {
            if (state.rows[i].images[j].id === imageId) {
                img = state.rows[i].images[j];
                state.rows[i].images.splice(j, 1);
                break;
            }
        }
        if (img) break;
    }
    if (!img) {
        for (var k = 0; k < state.staging_images.length; k++) {
            if (state.staging_images[k].id === imageId) {
                img = state.staging_images[k];
                state.staging_images.splice(k, 1);
                break;
            }
        }
    }
    if (!img) return;

    // Insert at target
    if (!targetRowId || targetRowId === 'null') {
        if (position < 0 || position >= state.staging_images.length) {
            state.staging_images.push(img);
        } else {
            state.staging_images.splice(position, 0, img);
        }
    } else {
        for (var i = 0; i < state.rows.length; i++) {
            if (state.rows[i].id === targetRowId) {
                if (position < 0 || position >= state.rows[i].images.length) {
                    state.rows[i].images.push(img);
                } else {
                    state.rows[i].images.splice(position, 0, img);
                }
                break;
            }
        }
    }
}

function removeImageFromState(imageId) {
    for (var i = 0; i < state.rows.length; i++) {
        for (var j = 0; j < state.rows[i].images.length; j++) {
            if (state.rows[i].images[j].id === imageId) {
                state.rows[i].images.splice(j, 1);
                return;
            }
        }
    }
    for (var k = 0; k < state.staging_images.length; k++) {
        if (state.staging_images[k].id === imageId) {
            state.staging_images.splice(k, 1);
            return;
        }
    }
}

// ========== Render Functions ==========

function renderAll() {
    document.querySelector('h1').textContent = state.title;
    renderRows();
    renderStaging();
}

function renderRows() {
    var container = document.getElementById('tier-container');
    // Remove all existing rows
    var existing = container.querySelectorAll('.tier-row');
    for (var i = 0; i < existing.length; i++) {
        existing[i].remove();
    }
    for (var i = 0; i < state.rows.length; i++) {
        container.appendChild(buildRowDOM(state.rows[i]));
    }
}

function renderStaging() {
    var staging = document.getElementById('staging-area');
    staging.innerHTML = '';
    for (var i = 0; i < state.staging_images.length; i++) {
        renderCharacter(state.staging_images[i], staging);
    }
}

function buildRowDOM(row) {
    var div = document.createElement('div');
    div.className = 'tier-row';
    div.setAttribute('data-row-id', row.id);
    div.innerHTML =
        '<div class="label-holder" style="background-color: ' + row.color + ';" data-color="' + row.color + '">' +
        '<span class="label">' + escapeHTML(row.label) + '</span></div>' +
        '<div class="tier sort"></div>' +
        '<div class="settings-control">' +
        '<div class="settings" title="设置">&#9881;</div>' +
        '<div class="move-buttons">' +
        '<div class="move-up">&#9650;</div>' +
        '<div class="move-down">&#9660;</div>' +
        '</div></div>';

    var sortZone = div.querySelector('.tier.sort');
    setupDropZone(sortZone);

    // Render images in this row
    for (var i = 0; i < row.images.length; i++) {
        renderCharacter(row.images[i], sortZone);
    }

    // Event bindings
    div.querySelector('.settings').addEventListener('click', function() {
        openSettings(div);
    });
    div.querySelector('.move-up').addEventListener('click', function() {
        send({type: 'move_row', row_id: row.id, direction: 'up'});
    });
    div.querySelector('.move-down').addEventListener('click', function() {
        send({type: 'move_row', row_id: row.id, direction: 'down'});
    });

    return div;
}

function renderCharacter(img, container) {
    var div = document.createElement('div');
    div.className = 'character';
    div.setAttribute('data-image-id', img.id);
    div.setAttribute('data-url', img.url);
    div.style.backgroundImage = 'url(' + img.url + ')';
    div.draggable = true;
    div.addEventListener('dragstart', handleDragStart);
    div.addEventListener('dragend', handleDragEnd);
    container.appendChild(div);
}

function removeRowDOM(rowId) {
    var row = document.querySelector('.tier-row[data-row-id="' + rowId + '"]');
    if (row) row.remove();
}

function removeImageDOM(imageId) {
    var el = document.querySelector('.character[data-image-id="' + imageId + '"]');
    if (el) el.remove();
}

function updateLabelDOM(rowId, label) {
    var row = document.querySelector('.tier-row[data-row-id="' + rowId + '"]');
    if (!row) return;
    var holder = row.querySelector('.label-holder');
    if (document.activeElement === holder || holder.contains(document.activeElement)) return;
    row.querySelector('.label').textContent = label;
    // Update modal textarea if this row is being edited and textarea is not focused
    if (activeRowId === rowId) {
        var ta = document.getElementById('labelName');
        if (document.activeElement !== ta) {
            ta.value = label;
        }
    }
}

function updateLabelColorDOM(rowId, color) {
    var row = document.querySelector('.tier-row[data-row-id="' + rowId + '"]');
    if (!row) return;
    var holder = row.querySelector('.label-holder');
    holder.style.backgroundColor = color;
    holder.setAttribute('data-color', color);
    // Update modal swatches if this row is being edited
    if (activeRowId === rowId) {
        highlightModalSwatch(color);
    }
}

function moveImageDOM(imageId, targetRowId, position) {
    var el = document.querySelector('.character[data-image-id="' + imageId + '"]');
    if (!el) return;

    var target;
    if (!targetRowId || targetRowId === 'null') {
        target = document.getElementById('staging-area');
    } else {
        var row = document.querySelector('.tier-row[data-row-id="' + targetRowId + '"]');
        if (!row) return;
        target = row.querySelector('.tier.sort');
    }

    var children = target.querySelectorAll('.character');
    if (position < 0 || position >= children.length) {
        target.appendChild(el);
    } else {
        target.insertBefore(el, children[position]);
    }
}

function moveImagesToStaging(rowId) {
    var row = document.querySelector('.tier-row[data-row-id="' + rowId + '"]');
    if (!row) return;
    var staging = document.getElementById('staging-area');
    var images = row.querySelectorAll('.tier.sort .character');
    for (var i = 0; i < images.length; i++) {
        staging.appendChild(images[i]);
    }
}

function moveAllImagesToStaging() {
    var staging = document.getElementById('staging-area');
    var rows = document.querySelectorAll('.tier-row');
    for (var i = 0; i < rows.length; i++) {
        var images = rows[i].querySelectorAll('.tier.sort .character');
        for (var j = 0; j < images.length; j++) {
            staging.appendChild(images[j]);
        }
    }
}

function highlightBgSwatch(color) {
    var swatches = document.querySelectorAll('#backgroundcolorselect span');
    for (var i = 0; i < swatches.length; i++) {
        swatches[i].classList.toggle('selected', normalizeColor(swatches[i].style.background) === normalizeColor(color));
    }
}

function highlightModalSwatch(color) {
    var swatches = document.querySelectorAll('#color-select span');
    for (var i = 0; i < swatches.length; i++) {
        swatches[i].classList.toggle('selected', normalizeColor(swatches[i].style.background) === normalizeColor(color));
    }
}

function normalizeColor(c) {
    if (!c) return '';
    return c.toLowerCase().replace(/\s/g, '');
}

function escapeHTML(str) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

// ========== Settings Modal ==========

function openSettings(row) {
    activeRowId = row.getAttribute('data-row-id');
    var labelHolder = row.querySelector('.label-holder');
    var curColor = labelHolder.getAttribute('data-color');
    var curLabel = labelHolder.querySelector('.label').textContent.trim();
    document.getElementById('labelName').value = curLabel;
    highlightModalSwatch(curColor);
    document.getElementById('overlay').classList.add('active');
}

function closeModal() {
    document.getElementById('overlay').classList.remove('active');
    activeRowId = null;
}

document.getElementById('overlay').addEventListener('click', function(e) {
    if (e.target === this) closeModal();
});

// Label textarea input: optimistic DOM update + debounced send
document.getElementById('labelName').addEventListener('input', function() {
    if (!activeRowId) return;
    var label = this.value || '?';
    // Optimistic: update the label span immediately
    var row = document.querySelector('.tier-row[data-row-id="' + activeRowId + '"]');
    if (row) {
        row.querySelector('.label').textContent = label;
    }
    clearTimeout(labelDebounceTimer);
    labelDebounceTimer = setTimeout(function() {
        send({type: 'update_label', row_id: activeRowId, label: label});
    }, 300);
});

function selectLabelColor(color, el) {
    if (!activeRowId) return;
    // Optimistic: update the label-holder immediately
    var row = document.querySelector('.tier-row[data-row-id="' + activeRowId + '"]');
    if (row) {
        var holder = row.querySelector('.label-holder');
        holder.style.backgroundColor = color;
        holder.setAttribute('data-color', color);
    }
    highlightModalSwatch(color);
    send({type: 'update_label_color', row_id: activeRowId, color: color});
}

function deleteRow() {
    if (!activeRowId) return;
    if (state.rows.length <= 1) {
        alert('不能删除最后一行。');
        return;
    }
    send({type: 'delete_row', row_id: activeRowId});
    closeModal();
}

function clearRow() {
    if (!activeRowId) return;
    send({type: 'clear_row', row_id: activeRowId});
}

function addRow(position) {
    if (!activeRowId) return;
    send({type: 'add_row', position: position, relative_to_row_id: activeRowId});
    closeModal();
}

function applyColorSequence() {
    if (!activeRowId) return;
    send({type: 'apply_color_sequence', row_id: activeRowId});
    closeModal();
}

// ========== Title Sync ==========

document.querySelector('h1').addEventListener('input', function() {
    clearTimeout(titleDebounceTimer);
    var self = this;
    titleDebounceTimer = setTimeout(function() {
        send({type: 'set_title', title: self.textContent.trim() || '未命名'});
    }, 300);
});

// ========== File Upload ==========

document.getElementById('file-input').addEventListener('change', function() {
    var files = this.files;
    if (!files.length) return;

    for (var i = 0; i < files.length; i++) {
        (function(file) {
            var reader = new FileReader();
            reader.onload = function(e) {
                // Strip data URL prefix to get raw base64
                var b64 = e.target.result;
                var commaIdx = b64.indexOf(',');
                if (commaIdx >= 0) {
                    b64 = b64.substring(commaIdx + 1);
                }
                send({
                    type: 'upload_image',
                    filename: file.name,
                    data: b64
                });
            };
            reader.readAsDataURL(file);
        })(files[i]);
    }
    this.value = '';
});

// ========== Drag and Drop ==========

function handleDragStart(e) {
    dragImageId = this.getAttribute('data-image-id');
    this.style.opacity = '0.5';
    e.dataTransfer.effectAllowed = 'move';
}

function handleDragEnd(e) {
    this.style.opacity = '1';
    dragImageId = null;
    var drops = document.querySelectorAll('.tier.sort, #staging-area, #delete-zone');
    for (var i = 0; i < drops.length; i++) {
        drops[i].classList.remove('drag-over');
    }
}

function allowDrop(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
}

// Drop on tier row sort zone
function handleSortDrop(e) {
    e.preventDefault();
    this.classList.remove('drag-over');
    if (!dragImageId) return;

    var targetRow = this.closest('.tier-row');
    var targetRowId = targetRow ? targetRow.getAttribute('data-row-id') : null;
    var position = computeDropIndex(this, e.clientX, e.clientY);
    var currentParent = document.querySelector('.character[data-image-id="' + dragImageId + '"]');
    var isSame = currentParent && currentParent.parentElement === this;

    send({
        type: 'move_image',
        image_id: dragImageId,
        target_row_id: targetRowId || 'null',
        target_index: position
    });

    // Optimistic: move the element immediately
    if (currentParent && !isSame) {
        var children = this.querySelectorAll('.character');
        if (position < 0 || position >= children.length) {
            this.appendChild(currentParent);
        } else {
            this.insertBefore(currentParent, children[position]);
        }
    }
}

// Drop on staging area
function handleStagingDrop(e) {
    e.preventDefault();
    this.classList.remove('drag-over');
    if (!dragImageId) return;

    var position = computeDropIndex(this, e.clientX, e.clientY);
    var currentParent = document.querySelector('.character[data-image-id="' + dragImageId + '"]');
    var isSame = currentParent && currentParent.parentElement === this;

    send({
        type: 'move_image',
        image_id: dragImageId,
        target_row_id: 'null',
        target_index: position
    });

    if (currentParent && !isSame) {
        var children = this.querySelectorAll('.character');
        if (position < 0 || position >= children.length) {
            this.appendChild(currentParent);
        } else {
            this.insertBefore(currentParent, children[position]);
        }
    }
}

function computeDropIndex(container, clientX, clientY) {
    var chars = container.querySelectorAll('.character');
    var idx = 0;
    for (var i = 0; i < chars.length; i++) {
        var rect = chars[i].getBoundingClientRect();
        var midX = rect.left + rect.width / 2;
        var midY = rect.top + rect.height / 2;
        if (clientX < midX && clientY < midY + rect.height / 2) {
            return i;
        }
        if (clientX < midX) {
            return i;
        }
        idx = i + 1;
    }
    return idx;
}

// Delete zone
document.getElementById('delete-zone').addEventListener('dragover', function(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    this.classList.add('drag-over');
});
document.getElementById('delete-zone').addEventListener('dragleave', function() {
    this.classList.remove('drag-over');
});
document.getElementById('delete-zone').addEventListener('drop', function(e) {
    e.preventDefault();
    this.classList.remove('drag-over');
    if (dragImageId) {
        send({type: 'delete_image', image_id: dragImageId});
        // Optimistic: remove from DOM
        var el = document.querySelector('.character[data-image-id="' + dragImageId + '"]');
        if (el) el.remove();
        dragImageId = null;
    }
});

// ========== Setup Drop Zones ==========

function setupDropZone(zone) {
    zone.addEventListener('dragover', function(e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        this.classList.add('drag-over');
    });
    zone.addEventListener('dragleave', function() {
        this.classList.remove('drag-over');
    });
    zone.addEventListener('drop', handleSortDrop);
}

// Staging area drop
document.getElementById('staging-area').addEventListener('dragover', allowDrop);
document.getElementById('staging-area').addEventListener('dragleave', function() {
    this.classList.remove('drag-over');
});
document.getElementById('staging-area').addEventListener('drop', handleStagingDrop);

// ========== Background Color ==========

function toggleBgColorPicker() {
    var el = document.getElementById('backgroundcolorselectContainer');
    el.style.display = el.style.display === 'none' ? 'block' : 'none';
}

function changeBgColor(color, el) {
    document.getElementById('tier-container').style.background = color;
    highlightBgSwatch(color);
}

// ========== Reset Button ==========

var resetTimer = null;
var resetStartTime = null;
var resetBtn = document.getElementById('reset-btn');
var resetProgress = document.getElementById('reset-progress');

function startReset(e) {
    e.preventDefault();
    if (resetTimer) return;
    resetStartTime = Date.now();
    resetBtn.classList.add('resetting');
    // Force animation restart
    resetProgress.style.animation = 'none';
    resetProgress.offsetHeight;
    resetProgress.style.animation = 'reset-fill 3s linear forwards';
    resetTimer = setTimeout(triggerReset, 3000);
}

function cancelReset() {
    if (!resetTimer) return;
    clearTimeout(resetTimer);
    resetTimer = null;
    resetStartTime = null;
    resetBtn.classList.remove('resetting');
    resetProgress.style.animation = 'none';
}

function triggerReset() {
    resetTimer = null;
    resetStartTime = null;
    resetBtn.classList.remove('resetting');
    resetBtn.classList.add('reset-done');
    resetProgress.style.animation = 'none';
    setTimeout(function() { resetBtn.classList.remove('reset-done'); }, 600);
    // Reset bg color locally
    document.getElementById('tier-container').style.background = '#1a1a1a';
    highlightBgSwatch('#1a1a1a');
    send({type: 'reset'});
}

if (resetBtn) {
    resetBtn.addEventListener('mousedown', startReset);
    resetBtn.addEventListener('mouseup', cancelReset);
    resetBtn.addEventListener('mouseleave', cancelReset);
    resetBtn.addEventListener('touchstart', startReset);
    resetBtn.addEventListener('touchend', cancelReset);
    resetBtn.addEventListener('touchcancel', cancelReset);
}

// ========== Stage All Button ==========

var stageAllTimer = null;
var stageAllBtn = document.getElementById('stage-all-btn');
var stageAllProgress = document.getElementById('stage-all-progress');

function startStageAll(e) {
    e.preventDefault();
    if (stageAllTimer) return;
    stageAllBtn.classList.add('resetting');
    stageAllProgress.style.animation = 'none';
    stageAllProgress.offsetHeight;
    stageAllProgress.style.animation = 'reset-fill 3s linear forwards';
    stageAllTimer = setTimeout(triggerStageAll, 3000);
}

function cancelStageAll() {
    if (!stageAllTimer) return;
    clearTimeout(stageAllTimer);
    stageAllTimer = null;
    stageAllBtn.classList.remove('resetting');
    stageAllProgress.style.animation = 'none';
}

function triggerStageAll() {
    stageAllTimer = null;
    stageAllBtn.classList.remove('resetting');
    stageAllBtn.classList.add('reset-done');
    stageAllProgress.style.animation = 'none';
    setTimeout(function() { stageAllBtn.classList.remove('reset-done'); }, 600);
    send({type: 'stage_all'});
}

if (stageAllBtn) {
    stageAllBtn.addEventListener('mousedown', startStageAll);
    stageAllBtn.addEventListener('mouseup', cancelStageAll);
    stageAllBtn.addEventListener('mouseleave', cancelStageAll);
    stageAllBtn.addEventListener('touchstart', startStageAll);
    stageAllBtn.addEventListener('touchend', cancelStageAll);
    stageAllBtn.addEventListener('touchcancel', cancelStageAll);
}

// ========== Page-Level File Drop ==========

function isFileDrag(e) {
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) return true;
    if (e.dataTransfer.types) {
        for (var i = 0; i < e.dataTransfer.types.length; i++) {
            if (e.dataTransfer.types[i] === 'Files') return true;
        }
    }
    if (e.dataTransfer.items) {
        for (var i = 0; i < e.dataTransfer.items.length; i++) {
            if (e.dataTransfer.items[i].kind === 'file') return true;
        }
    }
    return false;
}

document.addEventListener('dragover', function(e) {
    if (isFileDrag(e)) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'copy';
    }
}, true);

document.addEventListener('drop', function(e) {
    var files = e.dataTransfer.files;
    if (!files || !files.length) return;
    e.preventDefault();
    e.stopPropagation();

    for (var i = 0; i < files.length; i++) {
        (function(file) {
            var reader = new FileReader();
            reader.onload = function(e) {
                var b64 = e.target.result;
                var commaIdx = b64.indexOf(',');
                if (commaIdx >= 0) {
                    b64 = b64.substring(commaIdx + 1);
                }
                send({
                    type: 'upload_image',
                    filename: file.name,
                    data: b64
                });
            };
            reader.readAsDataURL(file);
        })(files[i]);
    }
});

// ========== Init ==========

connect();
