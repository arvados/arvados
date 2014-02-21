// This is a manifest file that'll be compiled into application.js, which will include all the files
// listed below.
//
// Any JavaScript/Coffee file within this directory, lib/assets/javascripts, vendor/assets/javascripts,
// or vendor/assets/javascripts of plugins, if any, can be referenced here using a relative path.
//
// It's not advisable to add code directly here, but if you do, it'll appear at the bottom of the
// the compiled file.
//
// WARNING: THE FIRST BLANK LINE MARKS THE END OF WHAT'S TO BE PROCESSED, ANY BLANK LINE SHOULD
// GO AFTER THE REQUIRES BELOW.
//
//= require jquery
//= require jquery_ujs
//= require bootstrap
//= require bootstrap/dropdown
//= require bootstrap/tab
//= require bootstrap/tooltip
//= require bootstrap/popover
//= require bootstrap/collapse
//= require bootstrap3-editable/bootstrap-editable
//= require_tree .

jQuery(function($){
    $.ajaxSetup({
        headers: {
            'X-CSRF-Token': $('meta[name="csrf-token"]').attr('content')
        }
    });
    $('.editable').editable();
    $('[data-toggle=tooltip]').tooltip();

    $('.expand-collapse-row').on('click', function(event) {
        var targets = $('#' + $(this).attr('data-id'));
        if (targets.css('display') == 'none') {
            $(this).addClass('icon-minus-sign');
            $(this).removeClass('icon-plus-sign');
        } else {
            $(this).addClass('icon-plus-sign');
            $(this).removeClass('icon-minus-sign');
        }
        targets.fadeToggle(200);
    });

    var add_selection = function(v) {
        var lst = JSON.parse(localStorage.persistentSelection);
        lst.push(v);
        localStorage.persistentSelection = JSON.stringify(lst);
        update_count();
    };

    var remove_selection = function(v) {
        var lst = JSON.parse(localStorage.persistentSelection);
        var i = jQuery.inArray(v, lst);
        if (i > -1) {
            lst.splice(i, 1);
        }
        localStorage.persistentSelection = JSON.stringify(lst);
        update_count();
    };

    var remove_selection_click = function(e) {
        remove_selection($(this).attr('name'));
    };

    var update_count = function(e) {
        var lst = JSON.parse(localStorage.persistentSelection);
        $("#persistent-selection-count").text(lst.length);

        if (lst.length > 0) {
            $('#persistent-selection-list').empty();
            for (var i = 0; i < lst.length; i++) {
                $('#persistent-selection-list').append("<li role=\"presentation\"><span><a href=\"#\">" + lst[i] + "</a>"
                                                       + "<a href=\"#\" class=\"remove-selection\" name=\"" + lst[i] + "\">" 
                                                       + "<span class=\"glyphicon glyphicon-trash pull-right\"></span>"
                                                       + "</a></span></li>");
            }
        } else {
            $('#persistent-selection-list').html("<li role=\"presentation\">No selections.</li>");
        }

        var checkboxes = $('.persistent-selection:checkbox');
        for (i = 0; i < checkboxes.length; i++) {
            if (jQuery.inArray($(checkboxes[i]).val(), lst) > -1) {
                checkboxes[i].checked = true;
            }
            else {
                checkboxes[i].checked = false;
            }
        }
        
        $('.remove-selection').on('click', remove_selection_click);
    };

    $(document).
        on('ajax:send', function(e, xhr) {
            $('.loading').fadeTo('fast', 1);
        }).
        on('ajax:complete', function(e, status) {
            $('.loading').fadeOut('fast', 0);
        }).
        on('change', '.persistent-selection:checkbox', function(e) {
            console.log($(this));
            console.log($(this).val());

            if (!localStorage.persistentSelection) {
                localStorage.persistentSelection = JSON.stringify([]);
            }
            
            var inc = 0;
            if ($(this).is(":checked")) {
                add_selection($(this).val());
            }
            else {
                remove_selection($(this).val());
            }
        });

    $(window).on('load storage', update_count);

    HeaderRowFixer = function(selector) {
        this.duplicateTheadTr = function() {
            $(selector).each(function() {
                var the_table = this;
                if ($('>tbody>tr:first>th', the_table).length > 0)
                    return;
                $('>tbody', the_table).
                    prepend($('>thead>tr', the_table).
                            clone().
                            css('opacity', 0));
            });
        }
        this.fixThead = function() {
            $(selector).each(function() {
                var widths = [];
                $('> tbody > tr:eq(1) > td', this).each( function(i,v){
                    widths.push($(v).width());
                });
                for(i=0;i<widths.length;i++) {
                    $('thead th:eq('+i+')', this).width(widths[i]);
                }
            });
        }
    }
    
    var fixer = new HeaderRowFixer('.table-fixed-header-row');
    fixer.duplicateTheadTr();
    fixer.fixThead();
    $(window).resize(function(){
        fixer.fixThead();
    });
    $(document).on('ajax:complete', function(e, status) {
        fixer.duplicateTheadTr();
        fixer.fixThead();
    });
})(jQuery);
