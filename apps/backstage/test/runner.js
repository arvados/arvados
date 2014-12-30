var global = (function() { return this })();

global.$ = global.jQuery = require('jquery');
chaiJquery = require('chai-jquery');
require('chai').use(chaiJquery);
mocha.setup({ui: 'tdd'});

require('test/unit/filter.js');
require('test/unit/filterset.js');
