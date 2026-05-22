// ========== Render Functions ==========

var _lastTapImageId = null;
var _lastTapTime = 0;
var _pendingTapImageId = null;
var _pendingTapTimer = null;

function renderAll() {
    document.querySelector('h1').textContent = state.title;
    renderRows();
    renderStaging();
}

function renderRows() {
    var container = document.getElementById('tier-container');
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

    for (var i = 0; i < row.images.length; i++) {
        renderCharacter(row.images[i], sortZone);
    }

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
    div.className = 'character' + (img.fit_width ? ' fit-width' : '');
    div.setAttribute('data-image-id', img.id);

    if (isMobileDevice) {
        setupMobileImageEvents(div, img);
    } else {
        setupDesktopImageEvents(div, img);
    }

    var imgEl = document.createElement('img');
    imgEl.src = img.url;
    imgEl.alt = '';
    div.appendChild(imgEl);

    container.appendChild(div);
}

function setupDesktopImageEvents(div, img) {
    div.draggable = true;
    div.addEventListener('dragstart', handleDragStart);
    div.addEventListener('dragend', handleDragEnd);
    div.addEventListener('contextmenu', function(e) {
        e.preventDefault();
        send({type: 'toggle_image_fit', image_id: img.id});
    });

    var pointerStart = null;
    div.addEventListener('pointerdown', function(e) {
        if (e.button === 0) pointerStart = {x: e.clientX, y: e.clientY};
    });
    div.addEventListener('pointerup', function(e) {
        if (pointerStart && Math.abs(e.clientX - pointerStart.x) < 4 && Math.abs(e.clientY - pointerStart.y) < 4) {
            openImageFullscreen(img.url);
        }
        pointerStart = null;
    });
}

function setupMobileImageEvents(div, img) {
    div.addEventListener('contextmenu', function(e) { e.preventDefault(); });
    div.addEventListener('selectstart', function(e) { e.preventDefault(); });

    var longPressTimer = null;
    var touchMoved = false;
    var longPressFired = false;
    var startX = 0;
    var startY = 0;

    div.addEventListener('touchstart', function(e) {
        e.preventDefault();
        var touch = e.touches[0];
        startX = touch.clientX;
        startY = touch.clientY;
        touchMoved = false;
        longPressFired = false;

        longPressTimer = setTimeout(function() {
            longPressTimer = null;
            longPressFired = true;
            _lastTapImageId = null;
            _lastTapTime = 0;
            if (navigator.vibrate) navigator.vibrate(15);
            send({type: 'toggle_image_fit', image_id: img.id});
        }, 500);
    }, {passive: false});

    div.addEventListener('touchmove', function(e) {
        if (!touchMoved && longPressTimer) {
            var touch = e.touches[0];
            if (Math.abs(touch.clientX - startX) > 8 || Math.abs(touch.clientY - startY) > 8) {
                touchMoved = true;
                clearTimeout(longPressTimer);
                longPressTimer = null;
            }
        }
    }, {passive: false});

    div.addEventListener('touchend', function(e) {
        e.preventDefault();
        if (longPressTimer) {
            clearTimeout(longPressTimer);
            longPressTimer = null;
        }
        if (touchMoved || longPressFired) return;

        var imageId = div.getAttribute('data-image-id');
        var now = Date.now();
        if (_lastTapImageId === imageId && (now - _lastTapTime) < 350) {
            _lastTapTime = 0;
            _lastTapImageId = null;
            if (_pendingTapTimer) {
                clearTimeout(_pendingTapTimer);
                _pendingTapTimer = null;
                _pendingTapImageId = null;
            }
            openImageFullscreen(img.url);
            return;
        }

        _lastTapImageId = imageId;
        _lastTapTime = now;

        if (_pendingTapTimer) {
            clearTimeout(_pendingTapTimer);
            _pendingTapTimer = null;
            _pendingTapImageId = null;
        }

        _pendingTapImageId = imageId;
        _pendingTapTimer = setTimeout(function() {
            _pendingTapTimer = null;
            if (selectedImageId === imageId) {
                deselectImage();
            } else {
                selectImage(imageId);
            }
            _pendingTapImageId = null;
        }, 200);
    }, {passive: false});

    div.addEventListener('touchcancel', function() {
        if (longPressTimer) {
            clearTimeout(longPressTimer);
            longPressTimer = null;
        }
    });
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
    if (activeRowId === rowId) {
        highlightModalSwatch(color);
    }
}

function moveImageDOM(imageId, targetRowId) {
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

    target.appendChild(el);
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
