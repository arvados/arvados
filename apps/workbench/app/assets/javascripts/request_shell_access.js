$(document).on('ready ajax:success storage', function() {
    // Update the "shell access requested" info box according to the
    // current state of localStorage.
    var msg = localStorage.getItem('request_shell_access');
    var $noShellAccessDiv = $('#no_shell_access');
    if ($noShellAccessDiv.length > 0) {
        $('.alert-success p', $noShellAccessDiv).text(msg);
        $('.alert-success', $noShellAccessDiv).toggle(!!msg);
    }
});
