// Usage:
//
// d = new Dict('examples');
// d.Put('one', [1,2,3])
// arr = d.Get('one');
module.exports.Dict = Dict;

function Dict(name) {
    this.name = name;
}

Dict.prototype.List = function() {
    return Object.keys(LoadDict(this.name));
}

Dict.prototype.Load = function() {
    return LoadDict(this.name);
}

Dict.prototype.Get = function(k) {
    return LoadDict(this.name)[k];
}

Dict.prototype.Put = function(k, v) {
    var data = LoadDict(this.name);
    data[k] = v;
    SaveDict(this.name, data);
    return v;
}

Dict.prototype.Delete = function(k) {
    var data = LoadDict(this.name);
    var v = data[k];
    delete(data[k])
    SaveDict(this.name, data);
    return v;
}

function LoadDict(key) {
    try {
        return JSON.parse(window.localStorage[key]);
    } catch(e) {
        return {};
    }
}

function SaveDict(key, data) {
    window.localStorage[key] = JSON.stringify(data);
}
