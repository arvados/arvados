$.fn.editable.defaults.ajaxOptions = {type: 'put', dataType: 'json'};
$.fn.editable.defaults.send = 'always';
$.fn.editable.defaults.params = function (params) {
    var a = {};
    var key = params.pk.key;
    a.id = params.pk.id;
    a[key] = {};
    a[key][params.name] = params.value;
    return a;
};