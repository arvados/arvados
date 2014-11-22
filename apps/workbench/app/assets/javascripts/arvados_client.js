angular.
    module('Arvados', []).
    service('ArvadosClient', ArvadosClient);

ArvadosClient.$inject = ['arvadosApiToken', 'arvadosDiscoveryUri']
function ArvadosClient(arvadosApiToken, arvadosDiscoveryUri) {
    $.extend(this, {
        apiPromise: apiPromise,
        uniqueNameForManifest: uniqueNameForManifest
    });
    return this;
    ////////////////////////////////

    var that = this;
    var promiseDiscovery;
    var discoveryDoc;

    function apiPromise(controller, action, params) {
        // Start an API call. Return a promise that will resolve with
        // the API response.
        return getDiscoveryDoc().then(function() {
            var meth = discoveryDoc.resources[controller].methods[action];
            var data = $.extend({}, params, {_method: meth.httpMethod});
            $.each(data, function(k, v) {
                if (typeof(v) == 'object') {
                    data[k] = JSON.stringify(v);
                }
            });
            var path = meth.path.replace(/{(.*?)}/, function(_, key) {
                var val = data[key];
                delete data[key];
                return encodeURIComponent(val);
            });
            return $.ajax({
                url: discoveryDoc.baseUrl + path,
                type: 'POST',
                crossDomain: true,
                dataType: 'json',
                data: data,
                headers: {
                    Authorization: 'OAuth2 ' + arvadosApiToken
                }
            });
        });
    }

    function uniqueNameForManifest(manifest, streamName, origName) {
        // Return an (escaped) filename starting with (unescaped)
        // origName that won't conflict with any existing names in
        // the manifest if saved under streamName. streamName must
        // be exactly as given in the manifest, e.g., "." or
        // "./foo" or "./foo/bar".
        //
        // Example:
        //
        // unique('./foo [...] 0:0:bar\040baz\n', '.', 'foo/bar baz')
        // =>
        // 'foo/bar\\040baz\\040(1)'
        var newName;
        var nameStub = origName;
        var suffixInt = null;
        var ok = false;
        while (!ok) {
            ok = true;
            // Add ' (N)' before the filename extension, if any.
            newName = (!suffixInt ? nameStub :
                       nameStub.replace(/(\.[^.]*)?$/, ' ('+suffixInt+')$1')).
                replace(/ /g, '\\040');
            $.each(manifest.split('\n'), function(_, line) {
                var i, match, foundName;
                var toks = line.split(' ');
                for (var i=1; i<toks.length && ok; i++)
                    if (match = toks[i].match(/^\d+:\d+:(\S+)/))
                        if (toks[0] + '/' + match[1] === streamName + '/' + newName) {
                            suffixInt = (suffixInt || 0) + 1;
                            ok = false;
                        }
            });
        }
        return newName;
    }

    function getDiscoveryDoc() {
        if (!promiseDiscovery) {
            promiseDiscovery = $.ajax({
                url: arvadosDiscoveryUri,
                crossDomain: true
            }).then(function(data, status, xhr) {
                discoveryDoc = data;
            });
        }
        return promiseDiscovery;
    }
}
