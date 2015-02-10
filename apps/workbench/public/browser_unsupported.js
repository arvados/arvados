(function() {
    var ok = false;
    try {
        if (window.Blob &&
            window.File &&
            window.FileReader &&
            window.WebSocket) {
            ok = true;
        }
    } catch(err) {}
    if (!ok) {
        document.getElementById('browser-unsupported').className='';
    }
})();
