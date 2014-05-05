$.fn.editable.defaults.ajaxOptions = {type: 'post', dataType: 'json'};
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
    a[key] = params.pk.defaults || {};
    a[key][params.name] = params.value;
    if (params.pk._method) {
        a['_method'] = params.pk._method;
    } else {
        a['_method'] = 'put';
    }
    return a;
};

$.fn.editable.defaults.validate = function (value) {
    if (value == "***invalid***") {
        return "Invalid selection";
    }
}

$(document).
    on('ready ajax:complete', function() {
        $('#editable-submit').click(function() {
            console.log($(this));
        });
        $('.editable').
            editable().
            on('hidden', function(e, reason) {
                if (reason == 'save') {
                    var html = $(this).html();
                    var uuid = $(this).attr('data-object-uuid');
                    var attr = $(this).attr('data-name');
                    var edited = this;
                    if (uuid && attr) {
                        $("[data-object-uuid='" + uuid + "']" +
                          "[data-name='" + attr + "']").each(function() {
                              if (this != edited)
                                  $(this).html(html);
                          });
                    }
                }
            });
    });

$.fn.editabletypes.text.defaults.tpl = '<input type="text" name="editable-text">'

$.fn.editableform.buttons = '\
<button type="submit" class="btn btn-primary btn-sm editable-submit" \
  id="editable-submit"><i class="glyphicon glyphicon-ok"></i></button>\
<button type="button" class="btn btn-default btn-sm editable-cancel" \
  id="editable-cancel"><i class="glyphicon glyphicon-remove"></i></button>\
'
