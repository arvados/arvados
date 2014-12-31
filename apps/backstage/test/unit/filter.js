var mq = require('mithril-query')
, Filter = require('app/filter')
, sinon = require('sinon')
, s = sinon;

suite('Filter', function() {
    function setup(filterClass) {
        var f = {};
        f.tested = new filterClass({attr: 'fakeAttr'});
        f.cfSpy = sinon.spy();
        f.ctrl = {currentFilter: f.cfSpy};
        f.rendered = f.tested.view(f.ctrl);
        f.domfrag = mq(f.rendered);
        return f;
    }
    suite('AnyText', function() {
        test.skip("default is existing filter value", function() {
            // TODO
        });
        test("fires currentFilter on input change", function() {
            f = setup(Filter.AnyText);
            // XXX: should call once to retrieve current filter value
            // s.assert.calledWith(f.cfSpy);
            f.domfrag.setValue('input', 'qux');
            // Should call again to set new filter value
            s.assert.calledWith(f.cfSpy, 'any', 'ilike', '%qux%');
        });
    });
    suite('ObjectType', function() {
        test.skip("default is existing filter value", function() {
            // TODO
        });
        test("fires currentFilter on selection", function() {
            f = setup(Filter.ObjectType);
            f.domfrag.click('li a');
            s.assert.calledOn(f.cfSpy, f.ctrl);
            s.assert.calledWith(f.cfSpy, 'fakeAttr', 'is_a', 'arvados#collection');
        });
    });
});
