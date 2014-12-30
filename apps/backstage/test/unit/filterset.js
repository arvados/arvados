var FilterSet = require('app/filterset')
, chai = require('chai')
, mq = require('mithril-query')
, sinon = require('sinon')
, s = sinon
, c = chai;

suite('FilterSet', function() {
    test("viewModules' views bind parentCtrl's currentFilter", function() {
        FilterStub = function() {};
        FilterStub.prototype.view = function() {};
        var stubFilterName = 'foo';
        var fakeFilterValue = 'bar';
        var childViewSpy = sinon.spy(FilterStub.prototype, 'view');
        var stubParentCtrl = {currentFilter: sinon.spy()};

        // Instantiate a component, and its controller.
        var tested = new FilterSet([[stubFilterName, FilterStub]]);
        var testedCtrl = new tested.controller(stubParentCtrl);

        // The FilterSet's view should invoke the child's view,
        // and pass a controller to it.
        s.assert.notCalled(childViewSpy);
        var viewOut = tested.view(testedCtrl);
        s.assert.calledOnce(childViewSpy);
        c.assert.lengthOf(childViewSpy.getCall(0).args, 1);

        // The controller passed to the child view should have a
        // currentFilter method which invokes
        // stubParentCtrl.currentFilter() with the filter name
        // prepended to its argument list.
        var childViewCtrl = childViewSpy.getCall(0).args[0];
        s.assert.notCalled(stubParentCtrl.currentFilter);
        childViewCtrl.currentFilter(fakeFilterValue);
        s.assert.calledOnce(stubParentCtrl.currentFilter);
        s.assert.calledOn(stubParentCtrl.currentFilter, stubParentCtrl);
        s.assert.calledWith(stubParentCtrl.currentFilter, stubFilterName, fakeFilterValue);
    });
});
