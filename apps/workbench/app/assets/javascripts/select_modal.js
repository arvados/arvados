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
    any = ($container.
           find('.selectable.active').length > 0)
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

}).on('click', '.modal button[data-action-href]', function() {
    var selection = [];
    var data = [];
    var $modal = $(this).closest('.modal');
    var action_data = $(this).data('action-data');
    var selection_param = action_data.selection_param;
    $modal.find('.modal-error').removeClass('hide').hide();
    $modal.find('.selectable.active[data-object-uuid]').each(function() {
        var val = $(this).attr('data-object-uuid');
        data.push({name: selection_param, value: val});
    });
    $.each(action_data, function(key, value) {
        data.push({name: key, value: value});
    });
    $.ajax($(this).attr('data-action-href'),
           {dataType: 'json',
            type: $(this).attr('data-method'),
            data: data,
            traditional: true,
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
    $(this).attr('href', '#');  // Skip normal click handler
    if ($(this).attr('data-project-uuid')) {
        params = {'filters[]': JSON.stringify(['owner_uuid',
                                               '=',
                                               $(this).attr('data-project-uuid')])};
    }
    // Use current selection as dropdown button label
    $(this).
        closest('.dropdown-menu').
        prev('button').
        html($(this).text() + ' <span class="caret"></span>');
    // Set (or unset) filter params and refresh filterable rows
    $($(this).closest('[data-filterable-target]').attr('data-filterable-target')).
        data('infinite-content-params', params).
        trigger('refresh-content');
});
$(document).on('page-refresh', function(event, data, status, jqxhr, action_data) {
    window.location.reload();
}).on('tab-refresh', function(event, data, status, jqxhr, action_data) {
    $(document).trigger('arv:pane:reload:all');
    $('body > .modal-container .modal').modal('hide');
}).on('redirect-to-created-object', function(event, data, status, jqxhr, action_data) {
    window.location.href = data.href.replace(/^[^\/]*\/\/[^\/]*/, '');
});
