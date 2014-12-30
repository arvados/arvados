require(['chai', 'test/webdriver-client'], function(chai, c) {
    var assert = chai.assert;
    describe('Dashboard page', function() {
        before(function() {
            c.init().url('http://localhost:5555');
        });
        it('has a nav', function() {
            assert(c.isVisible('nav'));
        });
    });
});
