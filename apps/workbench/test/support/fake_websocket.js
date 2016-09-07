sockets = [];
window.WebSocket = function(url) {
    sockets.push(this);
    window.setTimeout(function() {
        sockets.map(function(s) {
            s.onopen();
        });
        sockets.splice(0);
    }, 1);
}

window.WebSocket.prototype.send = function(msg) {
    // Uncomment for debugging:
    // console.log("fake WebSocket: send: "+msg);
}
