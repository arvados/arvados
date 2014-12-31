module.exports = mJquery;

var m = require('mithril');
var global = (function() { return this })();
var usingWin = global;
var ready;

mJquery.ready = function() {
    if (ready) {
        // Already done, or underway.
    } else if (typeof window !== 'undefined' && global === window) {
        ready = m.deferred();
        ready.resolve(require('jquery'));
    } else {
        ready = m.deferred();
        require('jsdom').env({
            html: '<!doctype html><html></html>',
            scripts: [
                '../jquery/dist/jquery.js',
                '../mithril/mithril.js',
            ],
            done: function(err, win) {
                if (err) {
                    console.log("jsdom setup failed: "+JSON.stringify(err));
                    ready.reject(err);
                } else {
                    usingWin = win;
                    ready.resolve(win.jQuery);
                }
            }
        });
    }
    return ready.promise;
}

function mJquery(selector, cell) {
    var $div = ready.promise()('<div></div>');
    (usingWin.m || m).render($div[0], cell);
    return ready.promise()(selector, $div[0]);
}
