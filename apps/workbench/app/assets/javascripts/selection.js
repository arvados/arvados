//= require jquery
//= require jquery_ujs

/** Javascript for local persistent selection. */

get_selection_list = null;

(function($){
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
        //remove_selection($(this).attr('name'));
        remove_selection($(this).val());
    };

    var clear_selections = function() {
        put_storage([]);
        update_count();
    }

    var update_count = function(e) {
        var lst = get_selection_list();
        $("#persistent-selection-count").text(lst.length);

        if (lst.length > 0) {
            $('#persistent-selection-list').html('<li><a href="#" class="btn pull-right" id="clear_selections_button">Clear selections</a></li>'
                                                 +'<li class="notification"><table style="width: 100%"></table></li>');
            for (var i = 0; i < lst.length; i++) {
                $('#persistent-selection-list > li > table').append("<tr>"
                                                       + "<td>"
                                                       + "<form>"
                                                       + "<input class='remove-selection' type='checkbox' value='" + lst[i].uuid + "' checked='true'></input>"
                                                       + "</form>"
                                                       + "</td>"

                                                       + "<td>"
                                                       + "<span style='padding-left: 1em'><a href=\"" + lst[i].href + "\">" + lst[i].name + "</a></span>"
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
})(jQuery);