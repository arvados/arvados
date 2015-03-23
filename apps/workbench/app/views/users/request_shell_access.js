var timestamp = new Date();
localStorage.setItem("request_shell_access",
                     "A request for shell access was sent on " +
                     timestamp.toLocaleDateString() +
                     " at " +
                     timestamp.toLocaleTimeString());
// The storage event gets triggered automatically in _other_ windows
// when we hit localStorage, but we also need to fire it manually in
// _this_ window.
$(document).trigger('storage');
