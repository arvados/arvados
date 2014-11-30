App.ApplicationRoute = Ember.Route.extend({
    model: function() {
        var that = this;
        $('body').
            addClass('arv-log-event-listener').
            attr('data-object-uuid', 'all').
            on('arv-log-event', function(event, data) {
                return that.updateStoreWithLogEvent(that.store, event, data);
            });
    },
    updateStoreWithLogEvent: function(store, event, data) {
        var payload = {};
        var attrs;
        var kind;
        var model;
        if (!(data &&
              data.object_kind &&
              data.event_type === 'update' &&
              data.properties &&
              data.properties.new_attributes)) {
            return;
        }
        kind = data.object_kind.replace('arvados#','');
        attrs = data.properties.new_attributes;
        model = store.getById(kind, attrs.uuid);
        if (!model) {
            // If the updated object isn't in our store, we don't care
            // about the update.
            return;
        }
        if (false) {
            // Unfortunately, attrs is just a hash of database values,
            // not the usual API response for this object type. If we
            // received the usual API response via websockets, we
            // would just do this:
            attrs.id = attrs.uuid;
            delete attrs.updated_at;
            payload[kind] = attrs;
            store.pushPayload(kind, payload);
        } else {
            // Until then, the safest approach is to ignore the detail
            // we get from data.properties.new_attributes and instead
            // use ember-data's ActiveRecordAdapter to reload the
            // model in a separate AJAX call.
            model.reload();
        }
    }
});
