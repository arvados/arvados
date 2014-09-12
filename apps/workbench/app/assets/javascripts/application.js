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
//= require bootstrap/modal
//= require bootstrap/button
//= require bootstrap3-editable/bootstrap-editable
//= require wiselinks
//= require_tree .

jQuery(function($){
    $.ajaxSetup({
        headers: {
            'X-CSRF-Token': $('meta[name="csrf-token"]').attr('content')
        }
    });

    $(document).ajaxStart(function(){
      $('.modal-with-loading-spinner .spinner').show();
    }).ajaxStop(function(){
      $('.modal-with-loading-spinner .spinner').hide();
    });

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

    var ajaxCount = 0;

    $(document).
        on('ajax:send', function(e, xhr) {
            ajaxCount += 1;
            if (ajaxCount == 1) {
                $('.loading').fadeTo('fast', 1);
            }
        }).
        on('ajax:complete', function(e, status) {
            ajaxCount -= 1;
            if (ajaxCount == 0) {
                $('.loading').fadeOut('fast', 0);
            }
        }).
        on('ajaxSend', function(e, xhr) {
            // jQuery triggers 'ajaxSend' event when starting an ajax call, but
            // rails-generated ajax triggers generate 'ajax:send'.  Workbench
            // event listeners currently expect 'ajax:send', so trigger the
            // rails event in response to the jQuery one.
            $(document).trigger('ajax:send');
        }).
        on('ajaxComplete', function(e, xhr) {
            // See comment above about ajaxSend/ajax:send
            $(document).trigger('ajax:complete');
        }).
        on('click', '.removable-tag a', function(e) {
            var tag_span = $(this).parents('[data-tag-link-uuid]').eq(0)
            tag_span.fadeTo('fast', 0.2);
            $.ajax('/links/' + tag_span.attr('data-tag-link-uuid'),
                   {dataType: 'json',
                    type: 'POST',
                    data: { '_method': 'DELETE' },
                    context: tag_span}).
                done(function(data, status, jqxhr) {
                    this.remove();
                }).
                fail(function(jqxhr, status, error) {
                    this.addClass('label-danger').fadeTo('fast', '1');
                });
            return false;
        }).
        on('click', 'a.add-tag-button', function(e) {
            var jqxhr;
            var new_tag_uuid = 'new-tag-' + Math.random();
            var tag_head_uuid = $(this).parents('tr').attr('data-object-uuid');
            var new_tag = window.prompt("Add tag for collection "+
                                    tag_head_uuid,
                                    "");
            if (new_tag == null)
                return false;
            var new_tag_span =
                $('<span class="label label-info removable-tag"></span>').
                attr('data-tag-link-uuid', new_tag_uuid).
                text(new_tag).
                css('opacity', '0.2').
                append('&nbsp;<span class="removable-tag"><a title="Delete tag"><i class="fa fa-fw fa-trash-o"></i></a></span>');
            $(this).
                parent().
                find('>span').
                append(new_tag_span).
                append(' ');
            $.ajax($(this).attr('data-remote-href'),
                           {dataType: 'json',
                            type: $(this).attr('data-remote-method'),
                            data: {
                                'link[head_uuid]': tag_head_uuid,
                                'link[link_class]': 'tag',
                                'link[name]': new_tag
                            },
                            context: new_tag_span}).
                done(function(data, status, jqxhr) {
                    this.attr('data-tag-link-uuid', data.uuid).
                        fadeTo('fast', '1');
                }).
                fail(function(jqxhr, status, error) {
                    this.addClass('label-danger').fadeTo('fast', '1');
                });
            return false;
        });

    $(document).
        on('ajax:complete ready', function() {
            // See http://getbootstrap.com/javascript/#buttons
            $('.btn').button();
        }).
        on('ready ajax:complete', function() {
            $('[data-toggle~=tooltip]').tooltip({container:'body'});
        }).
        on('ready ajax:complete', function() {
            // This makes the dialog close on Esc key, obviously.
            $('.modal').attr('tabindex', '-1')
        });

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

    $(document).ready(function() {
        /* When wiselinks is initialized, selection.js is not working. Since we want to stop
           using selection.js in the near future, let's not initialize wiselinks for now. */

        // window.wiselinks = new Wiselinks();

        $(document).off('page:loading').on('page:loading', function(event, $target, render, url){
            $("#page-wrapper").fadeOut(200);
        });

        $(document).off('page:redirected').on('page:redirected', function(event, $target, render, url){
        });

        $(document).off('page:always').on('page:always', function(event, xhr, settings){
            $("#page-wrapper").fadeIn(200);
        });

        $(document).off('page:done').on('page:done', function(event, $target, status, url, data){
        });

        $(document).off('page:fail').on('page:fail', function(event, $target, status, url, error, code){
        });
    });

    $(document).on('click', '.compute-detail', function(e) {
        console.log("woble");
        $(e.target).collapse('hide');
    });

});
