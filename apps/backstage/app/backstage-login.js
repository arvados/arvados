module.exports = BackstageLoginComponent;

var m = require('mithril');

function BackstageLoginComponent() {
    var callback = {};
    callback.controller = function() {
        var tokens = {};
        try {
            tokens = JSON.parse(window.localStorage.tokens);
        } catch(e) {}
        tokens[m.route.param('apiPrefix')] = m.route.param('api_token');
        window.localStorage.tokens = JSON.stringify(tokens);
        m.route(m.route.param('return_to') || '/');
    }
    callback.view = function(ctrl) {}
    return callback;
}
