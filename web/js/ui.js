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

// ========== Image Fullscreen ==========

function openImageFullscreen(url) {
    document.getElementById('image-fullscreen').src = url;
    document.getElementById('image-overlay').classList.add('active');
}

function closeImageFullscreen(e) {
    if (e && e.target !== document.getElementById('image-overlay')) return;
    document.getElementById('image-overlay').classList.remove('active');
}

document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        document.getElementById('image-overlay').classList.remove('active');
    }
});
