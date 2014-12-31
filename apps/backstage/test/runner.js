var global = (function() { return this })();

if (global.mocha) {
    // Running in browser.
    global.$ = global.jQuery = require('jquery');
    var cj = require('chai-jquery');
    var c = require('chai');
    c.use(cj);
    global.mocha.setup({ui: 'tdd'});
}

require('test/unit/filter.js');
require('test/unit/filterset.js');
