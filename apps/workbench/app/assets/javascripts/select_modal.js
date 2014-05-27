$(document).on('click', '.selectable', function() {
    var $this = $(this);
    if (!$this.hasClass('multiple')) {
        $this.closest('.selectable-container').
            find('.selectable').
            removeClass('active');
    }
    $this.toggleClass('active');
}).on('click', '.modal button[data-action-href]', function() {
    var selection = [];
    var data = {};
    var $modal = $(this).closest('.modal');
    $modal.find('.modal-error').removeClass('hide').hide();
    $modal.find('.selectable.active[data-object-uuid]').each(function() {
        selection.push($(this).attr('data-object-uuid'));
    });
    data[$(this).data('action-data').selection_param] = selection[0];
    $.ajax($(this).attr('data-action-href'),
           {dataType: 'json',
            type: $(this).attr('data-method'),
            data: data,
            context: {modal: $modal}}).
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
        success(function() {
            this.modal.find('.modal-error').hide();
            window.location.reload();
        });
});
