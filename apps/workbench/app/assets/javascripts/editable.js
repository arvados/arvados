$.fn.editable.defaults.ajaxOptions = {type: 'put', dataType: 'json'};
$.fn.editable.defaults.send = 'always';

// Default for editing is popup.  I experimented with inline which is a little
// nicer in that it shows up right under the mouse instead of nearby.  However,
// the inline box is taller than the regular content, which causes the page
// layout to shift unless we make the table rows tall, which leaves a lot of
// wasted space when not editing.  Also inline can get cut off if the page is
// too narrow, when the popup box will just move to do the right thing.
//$.fn.editable.defaults.mode = 'inline';

$.fn.editable.defaults.params = function (params) {
    var a = {};
    var key = params.pk.key;
    a.id = params.pk.id;
    a[key] = {};
    a[key][params.name] = params.value;
    return a;
};

$.fn.editable.defaults.validate = function (value) {
    if (value == "***invalid***") {
        return "Invalid selection";
    }
}
