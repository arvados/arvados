$.fn.editable.defaults.ajaxOptions = {type: 'put', dataType: 'json'};
$.fn.editable.defaults.send = 'always';
//$.fn.editable.defaults.mode = 'inline';
$.fn.editable.defaults.params = function (params) {
    var a = {};
    var key = params.pk.key;
    a.id = params.pk.id;
    a[key] = {};
    a[key][params.name] = params.value;
    return a;
};

(function() {
    $.fn.editable.defaults.success = function (response, newValue) {
        var tag = $(this);
        if (tag.hasClass("required")) {
            if (newValue && newValue.trim() != "") {
                tag.parent().css("background-color", "");
                tag.parent().prev().css("background-color", "");
            }
            else {
                tag.parent().css("background-color", "#ffdddd");
                tag.parent().prev().css("background-color", "#ffdddd");
            }
        }
    }

    $(window).on('load', function() {
        var a = $('a.editable.required');
        for (var i = 0; i < a.length; i++) {
            var tag = $(a[i]);
            if (tag.hasClass("editable-empty")) {
                tag.parent().css("background-color", "#ffdddd");
                tag.parent().prev().css("background-color", "#ffdddd");
            }
            else {
                tag.parent().css("background-color", "");
                tag.parent().prev().css("background-color", "");
            }
        }
    } );

})();
