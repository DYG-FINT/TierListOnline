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

document.getElementById('labelName').addEventListener('input', function() {
    if (!activeRowId) return;
    var label = this.value || '?';
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
        showToast('不能删除最后一行。', {type: 'error'});
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

// ========== Background Color ==========

function toggleBgColorPicker() {
    var el = document.getElementById('backgroundcolorselectContainer');
    el.style.display = el.style.display === 'none' ? 'block' : 'none';
}

function changeBgColor(color, el) {
    document.getElementById('tier-container').style.background = color;
    highlightBgSwatch(color);
}

// ========== Reset Button (now opens preset dialog) ==========

var resetBtn = document.getElementById('reset-btn');

if (resetBtn) {
    resetBtn.addEventListener('click', function(e) {
        e.preventDefault();
        send({type: 'list_presets'});
        openPresetDialog();
    });
}

// ========== Preset Selection Dialog ==========

var selectedPresetName = '';
var presetConfirmTimer = null;
var presetConfirmBtn = document.getElementById('preset-confirm-btn');
var presetConfirmProgress = document.getElementById('preset-confirm-progress');

function openPresetDialog() {
    document.getElementById('preset-overlay').classList.add('active');
    selectedPresetName = '';
    renderPresetList([]);
}

function closePresetDialog() {
    document.getElementById('preset-overlay').classList.remove('active');
    cancelPresetConfirm();
}

function renderPresetList(presets) {
    var list = document.getElementById('preset-list');
    list.innerHTML = '';

    var emptyBtn = createPresetItem('默认列表', '');
    emptyBtn.classList.add('selected');
    selectedPresetName = '';
    list.appendChild(emptyBtn);

    for (var i = 0; i < presets.length; i++) {
        var btn = createPresetItem(presets[i], presets[i]);
        list.appendChild(btn);
    }
}

function createPresetItem(label, name) {
    var btn = document.createElement('button');
    btn.className = 'preset-item';
    btn.textContent = label;
    btn.addEventListener('click', function() {
        var items = document.querySelectorAll('#preset-list .preset-item');
        for (var i = 0; i < items.length; i++) {
            items[i].classList.remove('selected');
        }
        btn.classList.add('selected');
        selectedPresetName = name;
    });
    return btn;
}

function updatePresetList(presets) {
    if (!presets || !presets.length) return;
    var list = document.getElementById('preset-list');
    var existing = list.querySelectorAll('.preset-item');
    // Remove all except the first ("空列表")
    for (var i = existing.length - 1; i > 0; i--) {
        existing[i].remove();
    }
    for (var i = 0; i < presets.length; i++) {
        var btn = createPresetItem(presets[i], presets[i]);
        list.appendChild(btn);
    }
}

document.getElementById('preset-overlay').addEventListener('click', function(e) {
    if (e.target === this) closePresetDialog();
});

document.getElementById('preset-close-btn').addEventListener('click', closePresetDialog);

// ========== Preset Confirm Long-Press ==========

function startPresetConfirm(e) {
    e.preventDefault();
    if (presetConfirmTimer) return;
    presetConfirmBtn.classList.add('resetting');
    presetConfirmProgress.style.animation = 'none';
    presetConfirmProgress.offsetHeight;
    presetConfirmProgress.style.animation = 'reset-fill 3s linear forwards';
    presetConfirmTimer = setTimeout(triggerPresetConfirm, 3000);
}

function cancelPresetConfirm() {
    if (!presetConfirmTimer) return;
    clearTimeout(presetConfirmTimer);
    presetConfirmTimer = null;
    if (presetConfirmBtn) {
        presetConfirmBtn.classList.remove('resetting');
        presetConfirmProgress.style.animation = 'none';
    }
}

function triggerPresetConfirm() {
    presetConfirmTimer = null;
    presetConfirmBtn.classList.remove('resetting');
    presetConfirmBtn.classList.add('reset-done');
    presetConfirmProgress.style.animation = 'none';
    setTimeout(function() { presetConfirmBtn.classList.remove('reset-done'); }, 600);
    send({type: 'load_preset', preset_name: selectedPresetName});
    closePresetDialog();
}

if (presetConfirmBtn) {
    presetConfirmBtn.addEventListener('mousedown', startPresetConfirm);
    presetConfirmBtn.addEventListener('mouseup', cancelPresetConfirm);
    presetConfirmBtn.addEventListener('mouseleave', cancelPresetConfirm);
    presetConfirmBtn.addEventListener('touchstart', startPresetConfirm);
    presetConfirmBtn.addEventListener('touchend', cancelPresetConfirm);
    presetConfirmBtn.addEventListener('touchcancel', cancelPresetConfirm);
}

// ========== Save Preset Dialog ==========

var savePresetBtn = document.getElementById('save-preset-btn');

if (savePresetBtn) {
    savePresetBtn.addEventListener('click', function() {
        openSavePresetDialog();
    });
}

function openSavePresetDialog() {
    document.getElementById('save-preset-input').value = '';
    document.getElementById('save-preset-overlay').classList.add('active');
    setTimeout(function() {
        document.getElementById('save-preset-input').focus();
    }, 100);
}

function closeSavePresetDialog() {
    document.getElementById('save-preset-overlay').classList.remove('active');
}

document.getElementById('save-preset-overlay').addEventListener('click', function(e) {
    if (e.target === this) closeSavePresetDialog();
});

document.getElementById('save-preset-close-btn').addEventListener('click', closeSavePresetDialog);

document.getElementById('save-preset-confirm-btn').addEventListener('click', function() {
    var input = document.getElementById('save-preset-input');
    var name = input.value.trim();
    if (!name) {
        showToast('请输入预设名称', {type: 'error'});
        return;
    }
    send({type: 'save_preset', preset_name: name});
});

document.getElementById('save-preset-input').addEventListener('keydown', function(e) {
    if (e.key === 'Enter') {
        document.getElementById('save-preset-confirm-btn').click();
    }
});

// ========== Add Text Dialog ==========

var addTextBtn = document.getElementById('add-text-btn');
var addTextConfirmBtn = document.getElementById('add-text-confirm-btn');
var addTextInput = document.getElementById('add-text-input');
var addTextPending = false;

if (addTextBtn) {
    addTextBtn.addEventListener('click', function() {
        openAddTextDialog();
    });
}

function openAddTextDialog() {
    addTextInput.value = '';
    addTextPending = false;
    addTextConfirmBtn.disabled = false;
    addTextInput.style.height = '';
    document.getElementById('add-text-overlay').classList.add('active');
    setTimeout(function() {
        addTextInput.focus();
    }, 100);
}

function closeAddTextDialog() {
    document.getElementById('add-text-overlay').classList.remove('active');
}

document.getElementById('add-text-overlay').addEventListener('click', function(e) {
    if (e.target === this) closeAddTextDialog();
});

document.getElementById('add-text-close-btn').addEventListener('click', closeAddTextDialog);

addTextConfirmBtn.addEventListener('click', function() {
    var content = addTextInput.value.trim();
    if (!content) {
        showToast('请输入文字内容', {type: 'error'});
        return;
    }
    if (content.length > 128) {
        showToast('文字长度超出限制', {type: 'error'});
        return;
    }
    addTextPending = true;
    addTextConfirmBtn.disabled = true;
    send({type: 'add_text', text_content: content});
});

addTextInput.addEventListener('keydown', function(e) {
    if (e.key === 'Enter' && !addTextPending) {
        addTextConfirmBtn.click();
    }
});

addTextInput.addEventListener('input', function() {
    this.style.height = '';
    this.style.height = Math.min(this.scrollHeight, 200) + 'px';
});

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

// ========== Image Fullscreen ==========

var currentFullscreenImageId = null;
var imageNameDebounceTimer = null;

function openImageFullscreen(url, name, imageId, displayType) {
    var imgEl = document.getElementById('image-fullscreen');
    imgEl.src = url || '';
    imgEl.style.display = displayType === 'text' ? 'none' : '';
    var nameBar = document.getElementById('image-name-bar');
    nameBar.textContent = name || '';
    nameBar.title = name || '';
    currentFullscreenImageId = imageId || null;
    var currentImg = findImageInState(imageId);
    highlightFullscreenSwatch(currentImg ? (currentImg.color || '') : '');
    document.getElementById('image-overlay').classList.add('active');
}

function selectFullscreenColor(color) {
    if (!currentFullscreenImageId) return;
    var currentImg = findImageInState(currentFullscreenImageId);
    var currentColor = currentImg ? (currentImg.color || '') : '';
    if (normalizeColor(color) === normalizeColor(currentColor)) return;

    var el = document.querySelector('.character[data-image-id="' + currentFullscreenImageId + '"]');
    if (el) el.style.backgroundColor = color || '';
    highlightFullscreenSwatch(color);

    send({type: 'set_image_color', image_id: currentFullscreenImageId, color: color});
}

function inverseColor(hex) {
    var h = hex.replace('#', '');
    var r = parseInt(h.substring(0, 2), 16);
    var g = parseInt(h.substring(2, 4), 16);
    var b = parseInt(h.substring(4, 6), 16);
    var ir = (255 - r).toString(16);
    if (ir.length < 2) ir = '0' + ir;
    var ig = (255 - g).toString(16);
    if (ig.length < 2) ig = '0' + ig;
    var ib = (255 - b).toString(16);
    if (ib.length < 2) ib = '0' + ib;
    return '#' + ir + ig + ib;
}

function selectFullscreenTextColor(color) {
    if (!currentFullscreenImageId) return;
    var currentImg = findImageInState(currentFullscreenImageId);
    var currentTextColor = currentImg && currentImg.text_color ? currentImg.text_color : '#ffffff';

    // Transparent → white, otherwise inverse
    var textColor;
    if (!color) {
        textColor = '#ffffff';
    } else {
        textColor = inverseColor(color);
    }

    if (normalizeColor(textColor) === normalizeColor(currentTextColor)) return;

    // Apply locally
    var el = document.querySelector('.character[data-image-id="' + currentFullscreenImageId + '"]');
    if (el) {
        var tcs = el.querySelector('.text-content');
        if (tcs) tcs.style.color = textColor;
    }

    send({type: 'set_image_text_color', image_id: currentFullscreenImageId, text_color: textColor});
    showToast('操作成功', {type: 'success'});
}

document.getElementById('image-name-bar').addEventListener('input', function() {
    if (!currentFullscreenImageId) return;
    clearTimeout(imageNameDebounceTimer);
    var nameBar = this;
    imageNameDebounceTimer = setTimeout(function() {
        send({type: 'rename_image', image_id: currentFullscreenImageId, image_name: nameBar.textContent.trim()});
    }, 300);
});

document.getElementById('image-name-bar').addEventListener('keydown', function(e) {
    if (e.key === 'Enter') {
        e.preventDefault();
    }
});

function closeImageFullscreen(e) {
    if (e && e.target !== document.getElementById('image-overlay')) return;
    document.getElementById('image-overlay').classList.remove('active');
    clearTimeout(imageNameDebounceTimer);
    currentFullscreenImageId = null;
}

document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        document.getElementById('image-overlay').classList.remove('active');
        clearTimeout(imageNameDebounceTimer);
        currentFullscreenImageId = null;
    }
});

var fullscreenColorPressTimer = null;
var fullscreenColorPressSpan = null;

document.getElementById('fullscreen-color-select').addEventListener('pointerdown', function(e) {
    var span = e.target.closest('span');
    if (!span) return;
    fullscreenColorPressSpan = span;
    fullscreenColorPressTimer = setTimeout(function() {
        fullscreenColorPressTimer = null;
        var color = fullscreenColorPressSpan.getAttribute('data-color') || '';
        selectFullscreenTextColor(color);
        fullscreenColorPressSpan = null;
    }, 500);
});

document.getElementById('fullscreen-color-select').addEventListener('pointerup', function(e) {
    if (fullscreenColorPressTimer) {
        clearTimeout(fullscreenColorPressTimer);
        fullscreenColorPressTimer = null;
        var span = e.target.closest('span');
        if (span) {
            var color = span.getAttribute('data-color') || '';
            selectFullscreenColor(color);
        }
        fullscreenColorPressSpan = null;
    }
});

document.getElementById('fullscreen-color-select').addEventListener('pointerleave', function() {
    if (fullscreenColorPressTimer) {
        clearTimeout(fullscreenColorPressTimer);
        fullscreenColorPressTimer = null;
        fullscreenColorPressSpan = null;
    }
});
