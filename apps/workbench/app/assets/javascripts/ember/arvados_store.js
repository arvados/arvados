App.ArvadosModel = DS.Model.extend({
    uuid: DS.attr('string'),
    createdAt: DS.attr('date'),
    etag: DS.attr('string'),
    href: DS.attr('string'),
    kind: DS.attr('string'),
    modifiedAt: DS.attr('date'),
    modifiedByClient_Uuid: DS.attr('string'),
    modifiedByUserUuid: DS.attr('string'),
    ownerUuid: DS.attr('string')
});

App.ApplicationAdapter = DS.ActiveModelAdapter.extend({
    host: $('meta[name=arvados-discovery-uri]').attr('content').
        replace(/\/discovery.*/, ''),
    namespace: 'arvados/v1',
    headers: {
        Authorization: 'OAuth2 '+$('meta[name=arvados-api-token]').attr('content')
    },
    defaultSerializer: 'arvados'
});

App.ArvadosSerializer = DS.ActiveModelSerializer.extend({
    extractArray: function(store, type, arvResp) {
        payload = {}
        if (arvResp.items)
            payload[Ember.Inflector.inflector.pluralize(type.typeKey)] = arvResp.items;
        else
            payload[type.typeKey] = arvResp;
        return this._super(store, type, payload);
    },
    extractSingle: function(store, primaryType, payload, recordId) {
        var rootedPayload = {}
        payload.id = payload.uuid;
        rootedPayload[payload.kind.replace('arvados#','')] = payload;
        return this._super(store, primaryType, rootedPayload, recordId);
    },
    normalize: function(type, hash, prop) {
        hash.id = hash.uuid;
        return this._super(type, hash, prop);
    }
});
