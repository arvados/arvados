//= require jquery
//= require jquery_ujs

/** Javascript for local persistent selection. */

get_selection_list = null;
form_selection_sources = {};

jQuery(function($){
    var storage = localStorage; // sessionStorage

    get_selection_list = function() {
        if (!storage.persistentSelection) {
            storage.persistentSelection = JSON.stringify([]);
        }
        return JSON.parse(storage.persistentSelection);
    }

    var put_storage = function(lst) {
        storage.persistentSelection = JSON.stringify(lst);
    }

    var add_selection = function(uuid, name, href, type) {
        var lst = get_selection_list();
        lst.push({"uuid": uuid, "name": name, "href": href, "type": type});
        put_storage(lst);
        update_count();
    };

    var remove_selection = function(uuid) {
        var lst = get_selection_list();
        for (var i = 0; i < lst.length; i++) {
            if (lst[i].uuid == uuid) {
                lst.splice(i, 1);
                i--;
            }
        }
        put_storage(lst);
        update_count();
    };

    var remove_selection_click = function(e) {
        remove_selection($(this).val());
    };

    var clear_selections = function() {
        put_storage([]);
        update_count();
    }

    var update_count = function(e) {
        var html;
        var this_object_uuid = $('#selection-form-content').
            closest('form').
            find('input[name=uuid]').val();
        var lst = get_selection_list();
        $("#persistent-selection-count").text(lst.length);
        if (lst.length > 0) {
            html = '<li><a href="#" class="btn btn-xs btn-info" id="clear_selections_button"><i class="fa fa-fw fa-ban"></i> Clear selections</a></li>';
            html += '<li><button class="btn btn-xs btn-info" type="submit" name="combine_selected_files_into_collection" '
                + ' id="combine_selected_files_into_collection">'
                + '<i class="fa fa-fw fa-archive"></i> Combine selected collections and files into a new collection</button></li>'
                + '<li class="notification"><table style="width: 100%"></table></li>';
            $('#selection-form-content').html(html);

            for (var i = 0; i < lst.length; i++) {
                $('#selection-form-content > li > table').append("<tr>"
                                                       + "<td>"
                                                       + "<input class='remove-selection' name='selection[]' type='checkbox' value='" + lst[i].uuid + "' checked='true' data-stoppropagation='true' />"
                                                       + "</td>"

                                                       + "<td>"
                                                       + "<div style='padding-left: 1em'><a href=\"" + lst[i].href + "\">" + lst[i].name + "</a></div>"
                                                       + "</td>"

                                                       + "<td style=\"vertical-align: top\">"
                                                       + "<span style=\"padding-right: 1em\">" + lst[i].type + "</span>"
                                                       + "</td>"

                                                       + "</tr>");
            }
        } else {
            $('#selection-form-content').html("<li class='notification empty'>No selections.</li>");
        }

        var checkboxes = $('.persistent-selection:checkbox');
        for (i = 0; i < checkboxes.length; i++) {
            for (var j = 0; j < lst.length; j++) {
                if (lst[j].uuid == $(checkboxes[i]).val()) {
                    checkboxes[i].checked = true;
                    break;
                }
            }
            if (j == lst.length) {
                checkboxes[i].checked = false;
            }
        }

        $('.remove-selection').on('click', remove_selection_click);
        $('#clear_selections_button').on('click', clear_selections);
        $(document).trigger('selections-updated', [lst]);
    };

    $(document).
        on('change', '.persistent-selection:checkbox', function(e) {
            var inc = 0;
            if ($(this).is(":checked")) {
                add_selection($(this).val(), $(this).attr('friendly_name'), $(this).attr('href'), $(this).attr('friendly_type'));
            }
            else {
                remove_selection($(this).val());
            }
        });

    $(window).on('load storage', update_count);

    $('#selection-form-content').on("click", function(e) {
        e.stopPropagation();
    });
});

add_form_selection_sources = null;
select_form_sources = null;

(function() {
    var form_selection_sources = {};
    add_form_selection_sources = function (src) {
        for (var i = 0; i < src.length; i++) {
            var t = form_selection_sources[src[i].type];
            if (!t) {
                t = form_selection_sources[src[i].type] = {};
            }
            if (!t[src[i].uuid]) {
                t[src[i].uuid] = src[i];
            }
        }
    };

    select_form_sources = function(type) {
        var ret = [];

        if (get_selection_list) {
            var lst = get_selection_list();
            if (lst.length > 0) {
                var text = "&horbar; Selections &horbar;";
                var span = document.createElement('span');
                span.innerHTML = text;
                ret.push({text: span.innerHTML, value: "***invalid***"});

                for (var i = 0; i < lst.length; i++) {
                    if (lst[i].type == type) {
                        var n = lst[i].name;
                        n = n.replace(/<span[^>]*>/i, "[");
                        n = n.replace(/<\/span>/i, "]");
                        ret.push({text: n, value: lst[i].uuid})
                    }
                }
            }
        }

        var text = "&horbar; Recent &horbar;";
        var span = document.createElement('span');
        span.innerHTML = text;
        ret.push({text: span.innerHTML, value: "***invalid***"});

        var t = form_selection_sources[type];
        for (var key in t) {
            if (t.hasOwnProperty(key)) {
                var obj = t[key];
                ret.push({text: obj.name, value: obj.uuid})
            }
        }
        return ret;
    };
})();

function dispatch_selection_action() {
    // Build a new "href" attribute for this link by starting with the
    // "data-href" attribute and appending ?foo[]=bar&foo[]=baz (or
    // &foo=... as appropriate) to reflect the current object
    // selections.
    var data = [];
    var param_name = $(this).attr('data-selection-param-name');
    var href = $(this).attr('data-href');
    if ($(this).closest('.disabled').length > 0) {
	return false;
    }
    $(this).
        closest('.selection-action-container').
        find(':checkbox:checked').
        each(function() {
            data.push({name: param_name, value: $(this).val()});
        });
    if (href.indexOf('?') >= 0)
        href += '&';
    else
        href += '?';
    href += $.param(data, true);
    $(this).attr('href', href);
    return true;
}

function enable_disable_selection_actions() {
    var $container = $(this).closest('.selection-action-container');
    var $checked = $('.persistent-selection:checkbox:checked', $container);
    $('[data-selection-action]').
        closest('div.btn-group-sm').
        find('ul li').
        toggleClass('disabled', ($checked.length == 0));
    $('[data-selection-action=compare]').
        closest('li').
        toggleClass('disabled',
                    ($checked.filter('[value*=-d1hrv-]').length < 2) ||
                    ($checked.not('[value*=-d1hrv-]').length > 0));
    $('[data-selection-action=copy]').
        closest('li').
        toggleClass('disabled',
                    ($checked.filter('[value*=-j7d0g-]').length > 0) ||
                    (($checked.not('[value*=-d1hrv-]').length > 0) && ($checked.filter('[value*=-]').length < 0)));
}

$(document).
    on('selections-updated ready ajax:complete', function() {
        var $btn = $('[data-selection-action]');
        $btn.click(dispatch_selection_action);
        enable_disable_selection_actions.call($btn);
    });
