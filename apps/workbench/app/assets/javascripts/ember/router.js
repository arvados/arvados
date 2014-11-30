App.Router.map(function() {
    this.resource('nodes', {path: '/'}, function() {
        this.resource('node', {path: ':id'});
    });
});
