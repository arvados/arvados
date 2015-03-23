$.fn.editable.defaults.ajaxOptions = {type: 'post', dataType: 'json'};
$.fn.editable.defaults.send = 'always';

// Default for editing is popup.  I experimented with inline which is a little
// nicer in that it shows up right under the mouse instead of nearby.  However,
// the inline box is taller than the regular content, which causes the page
// layout to shift unless we make the table rows tall, which leaves a lot of
// wasted space when not editing.  Also inline can get cut off if the page is
// too narrow, when the popup box will just move to do the right thing.
//$.fn.editable.defaults.mode = 'inline';

$.fn.editable.defaults.success = function (response, newValue) {
    $(document).trigger('editable:success', [this, response, newValue]);
};

$.fn.editable.defaults.params = function (params) {
    var a = {};
    var key = params.pk.key;
    a.id = $(this).attr('data-object-uuid') || params.pk.id;
    a[key] = params.pk.defaults || {};
    // Remove null values. Otherwise they get transmitted as empty
    // strings in request params.
    for (i in a[key]) {
        if (a[key][i] == null)
            delete a[key][i];
    }
    a[key][params.name] = params.value;
    if (!a.id) {
        a['_method'] = 'post';
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
        $('.editable').
            not('.editable-done-setup').
            addClass('editable-done-setup').
            editable({
                success: function(response, newValue) {
                    // If we just created a new object, stash its UUID
                    // so we edit it next time instead of creating
                    // another new object.
                    if (!$(this).attr('data-object-uuid') && response.uuid) {
                        $(this).attr('data-object-uuid', response.uuid);
                    }
                    if (response.href) {
                        $(this).editable('option', 'url', response.href);
                    }
                    if ($(this).attr('data-name')) {
                        var textileAttr = $(this).attr('data-name') + 'Textile';
                        if (response[textileAttr]) {
                            $(this).attr('data-textile', response[textileAttr]);
                        }
                    }
                    return;
                },
                error: function(response, newValue) {
                    var errlist = response.responseJSON.errors;
                    var errmsg;
                    if (Array.isArray(errlist)) {
                        errmsg = errlist.join();
                    } else {
                        errmsg = ("The server returned an error when making " +
                                  "this update (status " + response.status +
                                  ": " + errlist + ").");
                    }
                    return errmsg;
                }
            }).
            on('hidden', function(e, reason) {
                // After saving a new attribute, update the same
                // information if it appears elsewhere on the page.
                if (reason != 'save') return;
                var html = $(this).html();
                if( $(this).attr('data-textile') ) {
                    html = $(this).attr('data-textile');
                    $(this).html(html);
                }
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
            });
    }).
    on('ready ajax:complete', function() {
        $("[data-toggle~='x-editable']").
            not('.editable-done-setup').
            addClass('editable-done-setup').
            click(function(e) {
                e.stopPropagation();
                $($(this).attr('data-toggle-selector')).editable('toggle');
            });
    });

$.fn.editabletypes.text.defaults.tpl = '<input type="text" name="editable-text">'

$.fn.editableform.buttons = '\
<button type="submit" class="btn btn-primary btn-sm editable-submit" \
  id="editable-submit"><i class="glyphicon glyphicon-ok"></i></button>\
<button type="button" class="btn btn-default btn-sm editable-cancel" \
  id="editable-cancel"><i class="glyphicon glyphicon-remove"></i></button>\
'
