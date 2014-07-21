// Load tab panes on demand. See app/views/application/_content.html.erb

// Fire when a tab is selected/clicked. Check whether the content in
// the corresponding pane is loaded (or is being loaded). If not,
// start an AJAX request to load the content.
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

// Fire when the content in a tab pane becomes stale/dirty. If the
// pane is visible now, reload it right away. Otherwise, just replace
// the current content with a spinner for now: there's no need to load
// the new content unless/until the user clicks the corresponding tab.
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

// Mark all panes as stale/dirty. Refresh the active pane.
$(document).on('arv-log-event arv:pane:reload:all', function() {
    $('.tab-pane.loaded').trigger('arv:pane:reload');
});
