var toastIdCounter = 0;

function showToast(message, options) {
    options = options || {};
    var duration = options.duration || 3000;
    var type = options.type || 'info';

    var container = document.getElementById('toast-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'toast-container';
        document.body.appendChild(container);
    }

    var el = document.createElement('div');
    el.className = 'toast toast-' + type;
    el.textContent = message;
    el.style.animationDelay = '0s';

    container.insertBefore(el, container.firstChild);

    var timer = setTimeout(function() {
        dismiss(el);
    }, duration);

    el.addEventListener('mouseenter', function() {
        clearTimeout(timer);
    });
    el.addEventListener('mouseleave', function() {
        timer = setTimeout(function() {
            dismiss(el);
        }, duration);
    });

    el.addEventListener('animationend', function(e) {
        if (e.animationName === 'toast-out') {
            if (el.parentNode) {
                el.parentNode.removeChild(el);
            }
        }
    });

    function dismiss(el) {
        if (el.classList.contains('out')) return;
        el.classList.add('out');
    }
}