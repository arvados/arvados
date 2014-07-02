$(document).on('click', '.selectable', function() {
    var any;
    var $this = $(this);
    if (!$this.hasClass('multiple')) {
        $this.closest('.selectable-container').
            find('.selectable').
            removeClass('active');
    }
    $this.toggleClass('active');
    any = ($this.
           closest('.selectable-container').
           find('.selectable.active').length > 0)
    $this.
        closest('.modal').
        find('[data-enable-if-selection]').
        prop('disabled', !any);

    if ($this.hasClass('active')) {
        $(".modal-dialog-preview-pane").html('<div class="spinner spinner-32px spinner-h-center spinner-v-center"></div>');
        $.ajax($this.attr('data-preview-href'),
               {dataType: "html"}).
           done(function(data, status, jqxhr) {
                $(".modal-dialog-preview-pane").html(data);
            }).
            fail(function(data, status, jqxhr) {
                $(".modal-dialog-preview-pane").text('Preview load failed.');
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
});
$(document).on('page-refresh', function(event, data, status, jqxhr, action_data) {
    window.location.reload();
}).on('redirect-to-created-object', function(event, data, status, jqxhr, action_data) {
    window.location.href = data.href.replace(/^[^\/]*\/\/[^\/]*/, '');
});
