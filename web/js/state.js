// ========== Client State ==========

const state = {
    title: '',
    bg_color: '#1a1a1a',
    rows: [],
    staging_images: []
};

let activeRowId = null;
let dragImageId = null;
let labelDebounceTimer = null;
let titleDebounceTimer = null;

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

function moveImageInState(imageId, targetRowId) {
    var img = null;
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

    if (!targetRowId || targetRowId === 'null') {
        state.staging_images.push(img);
    } else {
        for (var i = 0; i < state.rows.length; i++) {
            if (state.rows[i].id === targetRowId) {
                state.rows[i].images.push(img);
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
