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
            $(this).trigger('arv:pane:loaded');
        }).fail(function(jqxhr, status, error) {
            var errhtml;
            if (jqxhr.getResponseHeader('Content-Type').match(/\btext\/html\b/)) {
                var $response = $(jqxhr.responseText);
                var $wrapper = $('div#page-wrapper', $response);
                if ($wrapper.length) {
                    errhtml = $wrapper.html();
                } else {
                    errhtml = jqxhr.responseText;
                }
            } else {
                errhtml = ("An error occurred: " +
                           (jqxhr.responseText || status)).
                    replace(/&/g, '&amp;').
                    replace(/</g, '&lt;').
                    replace(/>/g, '&gt;');
            }
            $('> div > div', this).html(
                '<div><p>' +
                    '<a href="#" class="btn btn-primary tab_reload">' +
                    '<i class="fa fa-fw fa-refresh"></i> ' +
                    'Reload tab</a></p><iframe></iframe></div>');
            $('.tab_reload', this).click(function() {
                $('> div > div', $pane).html(
                    '<div class="spinner spinner-32px spinner-h-center"></div>');
                $pane.trigger('arv:pane:reload');
            });
            // We want to render the error in an iframe, in order to
            // avoid conflicts with the main page's element ids, etc.
            // In order to do that dynamically, we have to set a
            // timeout on the iframe window to load our HTML *after*
            // the default source (e.g., about:blank) has loaded.
            var iframe = $('iframe', this)[0];
            iframe.contentWindow.setTimeout(function() {
                $('body', iframe.contentDocument).html(errhtml);
                iframe.height = iframe.contentDocument.body.scrollHeight + "px";
            }, 1);
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
