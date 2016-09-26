module.exports.Client = Client;
function Client(UUIDPrefix) {
    this.UUIDPrefix = UUIDPrefix;
}

Client.prototype.DiscoveryURL = function() {
    return 'https://' + this.UUIDPrefix + '.arvadosapi.com/discovery/v1/apis/arvados/v1/rest';
}

Client.prototype.LoginURL = function(callbackURL) {
    return 'https://' + this.UUIDPrefix + '.arvadosapi.com/login?return_to=' +
        encodeURIComponent(callbackURL);
}
