var mq = require('mithril-query')
, Filter = require('app/filter')
, sinon = require('sinon')
, s = sinon;

suite('Filter', function() {
    test("changing input fires currentFilter", function() {
        var tested = new Filter.AnyText();
        var cfSpy = sinon.spy();
        var ctrl = {currentFilter: cfSpy};
        var v = tested.view(ctrl);
        s.assert.notCalled(cfSpy);
        mq(v).setValue('input', 'qux');
        s.assert.calledOnce(cfSpy);
        s.assert.calledOn(cfSpy, ctrl);
        s.assert.calledWith(cfSpy, 'any', 'ilike', '%qux%');
    });
});
