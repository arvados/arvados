App.NodesRoute = Ember.Route.extend({
    model: function() {
        return this.store.find('node');
    }
});
