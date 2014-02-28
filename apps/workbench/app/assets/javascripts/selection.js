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

    var combine_selected_files_into_collection = function () {
        
    }

    var update_count = function(e) {
        var lst = get_selection_list();
        $("#persistent-selection-count").text(lst.length);

        if (lst.length > 0) {
            $('#selection-form-content').html('<li><input type="submit" name="combine_selected_files_into_collection" id="combine_selected_files_into_collection" value="Combine selected collections and files into a new collection"/></li>'
                                                 + '<li><a href="#" id="clear_selections_button">Clear selections</a></li>'
                                                 + '<li class="notification"><table style="width: 100%"></table></li>');
            for (var i = 0; i < lst.length; i++) {
                $('#selection-form-content > li > table').append("<tr>"
                                                       + "<td>"
                                                       + "<input class='remove-selection' name='selection[]' type='checkbox' value='" + lst[i].uuid + "' checked='true'></input>"
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
            $('#persistent-selection-list').html("<li class='notification empty'>No selections.</li>");
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
        $('#combine_selected_files_into_collection').on('click', combine_selected_files_into_collection);
    };

    $(document).
        on('change', '.persistent-selection:checkbox', function(e) {
            //console.log($(this));
            //console.log($(this).val());
            
            var inc = 0;
            if ($(this).is(":checked")) {
                add_selection($(this).val(), $(this).attr('friendly_name'), $(this).attr('href'), $(this).attr('friendly_type'));
            }
            else {
                remove_selection($(this).val());
            }
        });


    $(window).on('load storage', update_count);
});

add_form_selection_sources = null;
select_form_sources  = null;

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
                ret.push({text: "--- Selections ---", value: ""});

                for (var i = 0; i < lst.length; i++) {
                    if (lst[i].type == type) {
                        ret.push({text: lst[i].name, value: lst[i].uuid})
                    }
                }
            }
        }
        ret.push({text: "--- Recent ---", value: ""});

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

