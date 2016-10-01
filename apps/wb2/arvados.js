module.exports.Client = Client;
function Client(UUIDPrefix, token, requestFunc) {
    var cached = {};
    var dd;
    var client = {
        DiscoveryDoc: function() {
            if (dd === undefined)
                dd = requestFunc({
                    method: 'GET',
                    url: client.DiscoveryURL(),
                });
            return dd;
        },
        DiscoveryURL: function() {
            return 'https://' + UUIDPrefix + '.arvadosapi.com/discovery/v1/apis/arvados/v1/rest';
        },
        LoginURL: function(callbackURL) {
            return 'https://' + UUIDPrefix + '.arvadosapi.com/login?return_to=' +
                encodeURIComponent(callbackURL);
        },
        Get: function(path, params) {
            if (path in cached)
                return cached[path];
            var req = requestFunc({
                method: 'GET',
                url: 'https://' + UUIDPrefix + '.arvadosapi.com/arvados/v1/' + path,
                headers: [['Authorization', 'OAuth2 '+token]],
            });
            if (arguments.length == 1)
                cached[path] = req;
            return req;
        },
    };
    return client;
}
