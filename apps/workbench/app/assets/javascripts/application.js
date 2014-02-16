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
    $(document).
        on('ajax:send', function(e, xhr) {
            $('.loading').fadeTo('fast', 1);
        }).
        on('ajax:complete', function(e, status) {
            $('.loading').fadeOut('fast', 0);
        });

    HeaderRowFixer = function(selector) {
        var tables = $(selector);
        this.duplicateTheadTr = function() {
            tables.each(function() {
                var the_table = this;
                $('>tbody', the_table).
                    prepend($('>thead>tr', the_table).
                            clone().
                            css('opacity', 0));
            });
        }
        this.fixThead = function() {
            tables.each(function() {
                var widths = [];
                $('> tbody > tr:eq(0) > td', this).each( function(i,v){
                    widths.push($(v).width());
                });
                for(i=0;i<widths.length;i++) {
                    $('thead th:eq('+i+')', this).width(widths[i]);
                }
            });
        }
    }
    var fixer = new HeaderRowFixer('.table-fixed-header-row');
    fixer.fixThead();
    fixer.duplicateTheadTr();
    $(window).resize(function(){
        fixer.fixThead();
    });
})(jQuery);
