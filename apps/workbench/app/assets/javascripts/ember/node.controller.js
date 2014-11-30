App.NodesController = App.ArvadosArrayController.extend({
    itemController: 'node'
});

App.NodeController = App.ArvadosObjectController.extend({
    stateClassMap: {
        idle: 'label-success',
        alloc: 'label-primary',
        down: 'label-default'
    },
    stateClass: function() {
        var cws = this.get('model.crunchWorkerState');
        return this.stateClassMap[cws];
    }.property('model.crunchWorkerState'),

    dotDomain: function() {
        var domain = this.get('model.domain');
        return domain ? '.'+domain : '';
    }.property('model.domain')
});
