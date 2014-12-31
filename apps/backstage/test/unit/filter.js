var Filter = require('app/filter')
, chai = require('chai')
, m = require('mithril')
, md = require('test/mithril-dom')
, mq = require('mithril-query')
, sinon = require('sinon')
, $ = require('jquery')
, c = chai
, s = sinon;

suite('Filter', function() {
    function setup(filterClass, initialFilter) {
        var f = {};
        f.tested = new filterClass({attr: 'fakeAttr'});
        f.cfSpy = sinon.stub();
        f.cfSpy.withArgs().returns(initialFilter);
        f.ctrl = {currentFilter: f.cfSpy};
        f.rendered = f.tested.view(f.ctrl);
        f.domfrag = mq(f.rendered);
        return f;
    }
    suite('AnyText', function() {
        test("default is existing filter value", function() {
            f = setup(Filter.AnyText, ['any','ilike','%quux%']);
            f.rendered = f.tested.view(f.ctrl);
            f.domfrag = mq(f.rendered);
            c.assert.equal(f.domfrag.first('input').attrs.value, "quux");
        });
        test("fires currentFilter on input change", function() {
            f = setup(Filter.AnyText);
            f.domfrag.setValue('input', 'qux');
            // Should call again to set new filter value
            s.assert.calledWith(f.cfSpy, 'any', 'ilike', '%qux%');
        });
    });
    suite('ObjectType', function() {
        test("default is existing filter value", function() {
            f = setup(Filter.ObjectType, ['fakeAttr','is_a','arvados#collection']);
            f.rendered = f.tested.view(f.ctrl);
            f.md = md(f.rendered);
            c.assert.lengthOf($('.dropdown-toggle:contains(Type)', f.md), 0);
            c.assert.lengthOf($('.dropdown-toggle:contains(collection)', f.md), 1);
        });
        test("show generic label if no existing filter value", function() {
            f = setup(Filter.ObjectType, undefined);
            f.rendered = f.tested.view(f.ctrl);
            f.md = md(f.rendered);
            c.assert.lengthOf($('.dropdown-toggle:contains(Type)', f.md), 1);
        });
        test("fires currentFilter on selection", function() {
            f = setup(Filter.ObjectType);
            f.domfrag.click('li a[data-value="arvados#pipelineInstance"]');
            s.assert.calledOn(f.cfSpy, f.ctrl);
            s.assert.calledWith(f.cfSpy, 'fakeAttr', 'is_a', 'arvados#pipelineInstance');
        });
    });
});
