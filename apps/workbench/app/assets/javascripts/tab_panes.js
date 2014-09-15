// Load tab panes on demand. See app/views/application/_content.html.erb

// Fire when a tab is selected/clicked.
$(document).on('shown.bs.tab', '[data-toggle="tab"]', function(e) {
    $(this).trigger('arv:pane:reload');
});

// Fire when the content in a pane becomes stale/dirty. If the pane is
// 'active', reload it right away. Otherwise, just replace the current content
// with a spinner for now, don't load the new content unless/until the pane
// becomes active.
$(document).on('arv:pane:reload', function(e) {
    // Unload a single pane. Reload it if it's active.
    $(e.target).removeClass('loaded');
    var $pane = $($(e.target).attr('href'));
    if ($pane.hasClass('active')) {
        var content_url = $(e.target).attr('data-pane-content-url');
        $.ajax(content_url, {dataType: 'html', type: 'GET', context: $pane}).
            done(function(data, status, jqxhr) {
                // Preserve collapsed state
                var collapsable = {};
                $(".collapse", $pane).each(function(i, c) {
                    collapsable[c.id] = $(c).hasClass('in');
                });
                var tmp = $(data);
                $(".collapse", tmp).each(function(i, c) {
                    if (collapsable[c.id]) {
                        $(c).addClass('in');
                    } else {
                        $(c).removeClass('in');
                    }
                });
                $pane.html(tmp);
                $(e.target).addClass('loaded');
                $pane.trigger('arv:pane:loaded');
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
                $pane.html('<div><p>' +
                        '<a href="#" class="btn btn-primary tab_reload">' +
                        '<i class="fa fa-fw fa-refresh"></i> ' +
                        'Reload tab</a></p><iframe></iframe></div>');
                $('.tab_reload', $pane).click(function() {
                    $pane.html('<div class="spinner spinner-32px spinner-h-center"></div>');
                    $(e.target).trigger('arv:pane:reload');
                });
                // We want to render the error in an iframe, in order to
                // avoid conflicts with the main page's element ids, etc.
                // In order to do that dynamically, we have to set a
                // timeout on the iframe window to load our HTML *after*
                // the default source (e.g., about:blank) has loaded.
                var iframe = $('iframe', e.target)[0];
                iframe.contentWindow.setTimeout(function() {
                    $('body', iframe.contentDocument).html(errhtml);
                    iframe.height = iframe.contentDocument.body.scrollHeight + "px";
                }, 1);
                $(e.target).addClass('loaded');
            });
    } else {
        // When the user selects e.target tab, show a spinner instead of
        // old content while loading.
        $pane.html('<div class="spinner spinner-32px spinner-h-center"></div>');
    }
});

// Mark all panes as stale/dirty. Refresh the active pane.
$(document).on('arv-log-event arv:pane:reload:all', function() {
    $('.pane-anchor.loaded').trigger('arv:pane:reload');
});
