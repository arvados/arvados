(function() {
    var ok = false;
    try {
        if (window.Blob &&
            window.File &&
            window.FileReader) {
            ok = true;
        }
    } catch(err) {}
    if (!ok) {
        document.getElementById('browser-unsupported').className='';
    }
})();
