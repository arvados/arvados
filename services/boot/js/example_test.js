test = require('tape')
example = require('./example')
test('example is 42', function(t) {
    t.equal(42, example())
    t.end()
})
