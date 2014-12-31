var chai = require('chai')
, wd = require('webdriver-client')
, c = chai;

suite('Dashboard page', function() {
    setup(function() {
        wd.url('http://localhost:5555');
    });
    test('has a nav', function() {
        c.assert(wd.isVisible('nav'));
    });
});
