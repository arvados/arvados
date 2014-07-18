$(document).on('shown.bs.tab', '[data-toggle="tab"]', function(e) {
    var content_url = $(e.target).attr('data-pane-content-url');
    var $pane = $($(e.target).attr('href'));
    if ($pane.hasClass('loaded'))
        return;
    $.ajax(content_url, {dataType: 'html', type: 'GET', context: $pane}).
        done(function(data, status, jqxhr) {
            $('> div > div', this).html(data);
            $(this).addClass('loaded');
        });
});

$(document).on('arv:pane:reload', '.tab-pane', function() {
    // Unload a single pane. Reload it if it's active.
    $(this).removeClass('loaded');
    if ($(this).hasClass('active')) {
        $('[href=#' + $(this).attr('id') + ']').trigger('shown.bs.tab');
    } else {
        // When the user selects this tab, show a spinner instead of
        // old content while loading.
        $('> div > div', this).
            html('<div class="spinner spinner-32px spinner-h-center"></div>');
    }
});

$(document).on('arv-log-event arv:pane:reload:all', function() {
    // Reload all panes (except ones that haven't even loaded yet).
    $('.tab-pane.loaded').trigger('arv:pane:reload');
});
