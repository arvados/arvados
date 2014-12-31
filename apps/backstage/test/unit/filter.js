var Filter = require('app/filter')
, chai = require('chai')
, m = require('mithril')
, m$ = require('mithril-jquery')
, mq = require('mithril-query')
, sinon = require('sinon')
, $ = require('jquery')
, c = chai
, s = sinon;

suite('Filter', function() {
    setup(m$.ready);
    function prep(filterClass, initialFilter) {
        var f = {};
        f.tested = new filterClass({attr: 'fakeAttr'});
        f.cfSpy = sinon.stub();
        f.cfSpy.withArgs().returns(initialFilter);
        f.ctrl = {currentFilter: f.cfSpy};
        f.vdom = f.tested.view(f.ctrl);
        return f;
    }
    suite('AnyText', function() {
        test("uses existing filter as initial input value", function() {
            f = prep(Filter.AnyText, ['any','ilike','%quux%']);
            c.assert.equal(mq(f.vdom).first('input').attrs.value, "quux");
        });
        test("calls currentFilter when input changes", function() {
            f = prep(Filter.AnyText);
            mq(f.vdom).setValue('input', 'qux');
            // Should call again to set new filter value
            s.assert.calledWith(f.cfSpy, 'any', 'ilike', '%qux%');
        });
    });
    suite('ObjectType', function() {
        test("uses existing filter value as initial label", function() {
            f = prep(Filter.ObjectType, ['fakeAttr','is_a','arvados#collection']);
            c.assert.lengthOf(m$('.dropdown-toggle:contains(Type)', f.vdom), 0);
            c.assert.lengthOf(m$('.dropdown-toggle:contains(collection)', f.vdom), 1);
        });
        test("uses 'Type' as initial label if no current filter", function() {
            f = prep(Filter.ObjectType, undefined);
            c.assert.lengthOf(m$('.dropdown-toggle:contains(Type)', f.vdom), 1);
        });
        test("calls currentFilter when selection clicked", function() {
            f = prep(Filter.ObjectType);
            mq(f.vdom).click('li a[data-value="arvados#pipelineInstance"]');
            s.assert.calledOn(f.cfSpy, f.ctrl);
            s.assert.calledWith(f.cfSpy, 'fakeAttr', 'is_a', 'arvados#pipelineInstance');
        });
    });
});
