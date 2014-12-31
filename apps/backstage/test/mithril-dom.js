var $ = require('jquery')
, jsdom = require('jsdom')
, m = require('mithril');

module.exports = md;

var global = (function() { return this })();
var jsdomWin = global;

md.ready = function(cb) {
    if (typeof window !== 'undefined' && global === window) {
        cb($);
        return;
    }
    jsdom.env({
        html: '<html></html>',
        scripts: [
            '../node_modules/jquery/dist/jquery.js',
            '../node_modules/mithril/mithril.js',
        ],
        done: function(err, win) {
            jsdomWin = win;
            $ = win.jQuery;
            cb($);
        }
    });
}

function md(cell) {
    var div = $('<div></div>')[0];
    (jsdomWin.m || m).render(div, cell);
    return div.children;
}
