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

function handleSortDrop(e) {
    e.preventDefault();
    this.classList.remove('drag-over');
    if (!dragImageId) return;

    var targetRow = this.closest('.tier-row');
    var targetRowId = targetRow ? targetRow.getAttribute('data-row-id') : null;
    var currentParent = document.querySelector('.character[data-image-id="' + dragImageId + '"]');
    var isSame = currentParent && currentParent.parentElement === this;

    send({
        type: 'move_image',
        image_id: dragImageId,
        target_row_id: targetRowId || 'null',
        target_index: -1
    });

    if (currentParent && !isSame) {
        this.appendChild(currentParent);
    }
}

function handleStagingDrop(e) {
    e.preventDefault();
    this.classList.remove('drag-over');
    if (!dragImageId) return;

    var currentParent = document.querySelector('.character[data-image-id="' + dragImageId + '"]');
    var isSame = currentParent && currentParent.parentElement === this;

    send({
        type: 'move_image',
        image_id: dragImageId,
        target_row_id: 'null',
        target_index: -1
    });

    if (currentParent && !isSame) {
        this.appendChild(currentParent);
    }
}

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

    if (isMobileDevice) {
        zone.addEventListener('click', function(e) {
            if (!selectedImageId) return;
            if (e.target.closest('.character') || e.target.closest('.settings-control')) return;
            var currentEl = document.querySelector('.character[data-image-id="' + selectedImageId + '"]');
            if (currentEl && currentEl.parentElement === this) return;
            var targetRow = this.closest('.tier-row');
            var targetRowId = targetRow ? targetRow.getAttribute('data-row-id') : null;
            send({
                type: 'move_image',
                image_id: selectedImageId,
                target_row_id: targetRowId || 'null',
                target_index: -1
            });
            moveImageDOM(selectedImageId, targetRowId);
            deselectImage();
        });
    }
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
        var el = document.querySelector('.character[data-image-id="' + dragImageId + '"]');
        if (el) el.remove();
        dragImageId = null;
    }
});

// Staging area drop
document.getElementById('staging-area').addEventListener('dragover', allowDrop);
document.getElementById('staging-area').addEventListener('dragleave', function() {
    this.classList.remove('drag-over');
});
document.getElementById('staging-area').addEventListener('drop', handleStagingDrop);

// ========== Mobile Tap Handlers ==========

if (isMobileDevice) {
    document.getElementById('delete-zone').addEventListener('click', function(e) {
        if (!selectedImageId) return;
        if (e.target.closest('.character')) return;
        send({type: 'delete_image', image_id: selectedImageId});
        var el = document.querySelector('.character[data-image-id="' + selectedImageId + '"]');
        if (el) el.remove();
        deselectImage();
    });

    document.getElementById('staging-area').addEventListener('click', function(e) {
        if (!selectedImageId) return;
        if (e.target.closest('.character')) return;
        var currentEl = document.querySelector('.character[data-image-id="' + selectedImageId + '"]');
        if (currentEl && currentEl.parentElement === this) return;
        send({
            type: 'move_image',
            image_id: selectedImageId,
            target_row_id: 'null',
            target_index: -1
        });
        moveImageDOM(selectedImageId, 'null');
        deselectImage();
    });

    document.addEventListener('click', function(e) {
        if (!selectedImageId) return;
        if (e.target.closest('.character') ||
            e.target.closest('.tier.sort') ||
            e.target.closest('#staging-area') ||
            e.target.closest('#delete-zone') ||
            e.target.closest('.settings-control') ||
            e.target.closest('button') ||
            e.target.closest('label') ||
            e.target.closest('#image-overlay') ||
            e.target.closest('#overlay') ||
            e.target.closest('h1')) {
            return;
        }
        deselectImage();
    });
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
