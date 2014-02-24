//= require jquery
//= require jquery_ujs

/** Javascript for local persistent selection. */

(function($){
    var get_storage = function() {
        if (!sessionStorage.persistentSelection) {
            sessionStorage.persistentSelection = JSON.stringify([]);
        }
        return JSON.parse(sessionStorage.persistentSelection);
    }

    var put_storage = function(lst) {
        sessionStorage.persistentSelection = JSON.stringify(lst);
    }

    var add_selection = function(uuid, name, href, type) {
        var lst = get_storage();
        lst.push({"uuid": uuid, "name": name, "href": href, "type": type});
        put_storage(lst);
        update_count();
    };

    var remove_selection = function(uuid) {
        var lst = get_storage();
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
        remove_selection($(this).attr('name'));
    };

    var update_count = function(e) {
        var lst = get_storage();
        $("#persistent-selection-count").text(lst.length);

        $('#persistent-selection-list > li > table').empty();
        if (lst.length > 0) {
            for (var i = 0; i < lst.length; i++) {
                $('#persistent-selection-list > li > table').append("<tr>"
                                                       + "<td style=\"vertical-align: top\">"
                                                       + "<span style=\"padding-right: 1em\">" + lst[i].type + "</span>"
                                                       + "</td>"

                                                       + "<td>"
                                                       + "<span><a href=\"" + lst[i].href + "\">" + lst[i].name + "</a></span>"
                                                       + "</td>"

                                                       + "<td>"
                                                       + "<a href=\"#\" class=\"remove-selection\" name=\"" + lst[i].uuid + "\">" 
                                                       + "<span class=\"glyphicon glyphicon-trash pull-right\"></span>"
                                                       + "</a></span>"
                                                       + "</td>"
                                                       + "</tr>");
            }
        } else {
            $('#persistent-selection-list > li > table').html("<tr><td>No selections.</td></tr>");
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