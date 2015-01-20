$(document).on('click', '.selectable', function() {
    var any;
    var $this = $(this);
    var $container = $(this).closest('.selectable-container');
    if (!$container.hasClass('multiple')) {
        $container.
            find('.selectable').
            removeClass('active');
    }
    $this.toggleClass('active');

    if (!$this.hasClass('use-preview-selection')) {
      any = ($container.
           find('.selectable.active').length > 0)
    }

    if (!$container.hasClass('preview-selectable-container')) {
      $this.
        closest('.modal').
        find('[data-enable-if-selection]').
        prop('disabled', !any);

      if ($this.hasClass('active')) {
        var no_preview_available = '<div class="spinner-h-center spinner-v-center"><center>(No preview available)</center></div>';
        if (!$this.attr('data-preview-href')) {
            $(".modal-dialog-preview-pane").html(no_preview_available);
            return;
        }
        $(".modal-dialog-preview-pane").html('<div class="spinner spinner-32px spinner-h-center spinner-v-center"></div>');
        $.ajax($this.attr('data-preview-href'),
               {dataType: "html"}).
            done(function(data, status, jqxhr) {
                $(".modal-dialog-preview-pane").html(data);
            }).
            fail(function(data, status, jqxhr) {
                $(".modal-dialog-preview-pane").html(no_preview_available);
            });
      }
    } else {
      any = ($container.
           find('.preview-selectable.active').length > 0)
      $(this).
          closest('.modal').
          find('[data-enable-if-selection]').
          prop('disabled', !any);
    }

}).on('click', '.modal button[data-action-href]', function() {
    var selection = [];
    var data = [];
    var $modal = $(this).closest('.modal');
    var http_method = $(this).attr('data-method').toUpperCase();
    var action_data = $(this).data('action-data');
    var action_data_from_params = $(this).data('action-data-from-params');
    var selection_param = action_data.selection_param;
    $modal.find('.modal-error').removeClass('hide').hide();

    var $preview_selections = $modal.find('.preview-selectable.active');
    if ($preview_selections.length > 0) {
      data.push({name: selection_param, value: $preview_selections.first().attr('href')});
    }

    if (data.length == 0) {   // not using preview selection option
      $modal.find('.selectable.active[data-object-uuid]').each(function() {
        var val = $(this).attr('data-object-uuid');
        data.push({name: selection_param, value: val});
      });
    }
    $.each($.extend({}, action_data, action_data_from_params),
           function(key, value) {
               if (value instanceof Array && key[-1] != ']') {
                   for (var i in value) {
                       data.push({name: key + '[]', value: value[i]});
                   }
               } else {
                   data.push({name: key, value: value});
               }
           });
    if (http_method === 'PATCH') {
        // Some user agents do not support HTTP PATCH (notably,
        // phantomjs silently ignores our "data" and sends an empty
        // request body) so we use POST instead, and supply a
        // _method=PATCH param to tell Rails what we really want.
        data.push({name: '_method', value: http_method});
        http_method = 'POST';
    }
    $.ajax($(this).attr('data-action-href'),
           {dataType: 'json',
            type: http_method,
            data: data,
            traditional: false,
            context: {modal: $modal, action_data: action_data}}).
        fail(function(jqxhr, status, error) {
            if (jqxhr.readyState == 0 || jqxhr.status == 0) {
                message = "Cancelled."
            } else if (jqxhr.responseJSON && jqxhr.responseJSON.errors) {
                message = jqxhr.responseJSON.errors.join("; ");
            } else {
                message = "Request failed.";
            }
            this.modal.find('.modal-error').
                html('<div class="alert alert-danger">' + message + '</div>').
                show();
        }).
        done(function(data, status, jqxhr) {
            var event_name = this.action_data.success;
            this.modal.find('.modal-error').hide();
            $(document).trigger(event_name!=null ? event_name : 'page-refresh',
                                [data, status, jqxhr, this.action_data]);
        });
}).on('click', '.chooser-show-project', function() {
    var params = {};
    var project_uuid = $(this).attr('data-project-uuid');
    $(this).attr('href', '#');  // Skip normal click handler
    if (project_uuid) {
        params = {'filters': [['owner_uuid',
                               '=',
                               project_uuid]],
                  'project_uuid': project_uuid
                 };
    }
    // Use current selection as dropdown button label
    $(this).
        closest('.dropdown-menu').
        prev('button').
        html($(this).text() + ' <span class="caret"></span>');
    // Set (or unset) filter params and refresh filterable rows
    $($(this).closest('[data-filterable-target]').attr('data-filterable-target')).
        data('infinite-content-params-from-project-dropdown', params).
        trigger('refresh-content');
}).on('ready', function() {
    $('form[data-search-modal] a').on('click', function() {
        $(this).closest('form').submit();
        return false;
    });
    $('form[data-search-modal]').on('submit', function() {
        // Ask the server for a Search modal. When it arrives, copy
        // the search string from the top nav input into the modal's
        // search query field.
        var $form = $(this);
        var searchq = $form.find('input').val();
        var is_a_uuid = /^([0-9a-f]{32}(\+\S+)?|[0-9a-z]{5}-[0-9a-z]{5}-[0-9a-z]{15})$/;
        if (searchq.trim().match(is_a_uuid)) {
            window.location = '/actions?uuid=' + encodeURIComponent(searchq.trim());
            // Show the "loading" indicator. TODO: better page transition hook
            $(document).trigger('ajax:send');
            return false;
        }
        if ($form.find('a[data-remote]').length > 0) {
            // A search dialog is already loading.
            return false;
        }
        $('<a />').
            attr('href', $form.attr('data-search-modal')).
            attr('data-remote', 'true').
            attr('data-method', 'GET').
            hide().
            appendTo($form).
            on('ajax:success', function(data, status, xhr) {
                $('body > .modal-container input[type=text]').
                    val($form.find('input').val()).
                    focus();
                $form.find('input').val('');
            }).on('ajax:complete', function() {
                $(this).detach();
            }).
            click();
        return false;
    });
}).on('page-refresh', function(event, data, status, jqxhr, action_data) {
    window.location.reload();
}).on('tab-refresh', function(event, data, status, jqxhr, action_data) {
    $(document).trigger('arv:pane:reload:all');
    $('body > .modal-container .modal').modal('hide');
}).on('redirect-to-created-object', function(event, data, status, jqxhr, action_data) {
    window.location.href = data.href.replace(/^[^\/]*\/\/[^\/]*/, '');
}).on('shown.bs.modal', 'body > .modal-container .modal', function() {
    $('.focus-on-display', this).focus();
});
