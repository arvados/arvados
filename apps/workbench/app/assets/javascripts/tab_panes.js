// Load tab panes on demand. See app/views/application/_content.html.erb

// Fire when a tab is selected/clicked.
$(document).on('shown.bs.tab', '[data-toggle="tab"]', function(event) {
    // When we switch tabs, remove "active" from any refreshable panes within
    // the previous tab content so they don't continue to refresh unnecessarily, and
    // add "active" to any refreshable panes under the newly shown tab content.

    var tgt = $($(event.relatedTarget).attr('href'));
    $(".pane-anchor", tgt).each(function (i, e) {
        var a = $($(e).attr('href'));
        a.removeClass("active");
    });

    tgt = $($(event.target).attr('href'));
    $(".pane-anchor", tgt).each(function (i, e) {
        var a = $($(e).attr('href'));
        a.addClass("active");
    });

    // Now trigger reload of the newly shown tab pane.
    $(event.target).trigger('arv:pane:reload');
});

// Ask a refreshable pane to reload via ajax.
// Target of this event is the anchoring element that manages the pane.
//
// Panes can be in one of three primary states, managed by setting CSS classes
// on the object: not loaded (no pane-* state classes), pane-loading, pane-loaded
//
// not loaded means the pane needs to be loaded when the pane becomes active
//
// pane-loading means there is a current AJAX call outstanding to reload the pane
//
// pane-loaded means the pane is believe to be up to date
//
// There are two additional states: pane-stale and pane-reload-pending
//
// pane-stale indicates a pane that is already loading has been invalidated and
// should schedule a reload immediately when the current load completes.  (This
// happens if there are clusters of events, where the reload is trigged by the
// first event, but we actually want to display the state after the final event
// has been processed.)
//
// pane-reload-pending indicates a reload is scheduled, to suppress
// scheduling any additional reloads.
$(document).on('arv:pane:reload', function(e) {
    e.stopPropagation();

    // '$anchor' is the event target, which is a .pane-anchor or a bootstrap
    // tab anchor.  This is the element that stores the state of the pane.  The
    // actual element that will contain the content is pointed to in the 'href'
    // attribute of etarget.
    var $anchor = $(e.target);

    if ($anchor.hasClass('pane-loading')) {
        // Already loading, mark stale to schedule a reload after this one.
        $anchor.addClass('pane-stale');
        return;
    }

    if ($anchor.hasClass('pane-no-auto-reload') && $anchor.hasClass('pane-loaded')) {
        // Have to explicitly remove pane-loaded if we want it to reload.
        return;
    }

    var throttle = $anchor.attr('data-load-throttle');
    if (!throttle) {
        throttle = 3000;
    }
    var now = (new Date()).getTime();
    var loaded_at = $anchor.attr('data-loaded-at');
    var since_last_load = now - loaded_at;
    if (loaded_at && (since_last_load < throttle)) {
        if (!$anchor.hasClass('pane-reload-pending')) {
            $anchor.addClass('pane-reload-pending');
            setTimeout((function() {
                $anchor.trigger('arv:pane:reload');
            }), throttle - since_last_load);
        }
        return;
    }

    // We know this doesn't have 'pane-loading' because we tested for it above
    $anchor.removeClass('pane-reload-pending');
    $anchor.removeClass('pane-loaded');
    $anchor.removeClass('pane-stale');

    // $pane is the actual content area that is going to be updated.
    var $pane = $($anchor.attr('href'));
    if ($pane.hasClass('active')) {
        $anchor.addClass('pane-loading');

        var content_url = $anchor.attr('data-pane-content-url');
        $.ajax(content_url, {dataType: 'html', type: 'GET', context: $pane}).
            done(function(data, status, jqxhr) {
                // Preserve collapsed state
                var collapsable = {};
                $(".collapse", $(this)).each(function(i, c) {
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
                $(this).html(tmp);
                $anchor.removeClass('pane-loading');
                $anchor.addClass('pane-loaded');
                $anchor.attr('data-loaded-at', (new Date()).getTime());
                $(this).trigger('arv:pane:loaded');

                if ($anchor.hasClass('pane-stale')) {
                    $anchor.trigger('arv:pane:reload');
                }
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
                $(this).html('<div><p>' +
                        '<a href="#" class="btn btn-primary tab_reload">' +
                        '<i class="fa fa-fw fa-refresh"></i> ' +
                        'Reload tab</a></p><iframe style="width: 100%"></iframe></div>');
                $('.tab_reload', $(this)).click(function() {
                    $(this).html('<div class="spinner spinner-32px spinner-h-center"></div>');
                    $anchor.trigger('arv:pane:reload');
                });
                // We want to render the error in an iframe, in order to
                // avoid conflicts with the main page's element ids, etc.
                // In order to do that dynamically, we have to set a
                // timeout on the iframe window to load our HTML *after*
                // the default source (e.g., about:blank) has loaded.
                var iframe = $('iframe', $(this))[0];
                iframe.contentWindow.setTimeout(function() {
                    $('body', iframe.contentDocument).html(errhtml);
                    iframe.height = iframe.contentDocument.body.scrollHeight + "px";
                }, 1);
                $anchor.addClass('pane-loaded');
            });
    } else {
        // When the user selects e.target tab, show a spinner instead of
        // old content while loading.
        $pane.html('<div class="spinner spinner-32px spinner-h-center"></div>');
    }
});

// Mark all panes as stale/dirty. Refresh any 'active' panes.
$(document).on('arv:pane:reload:all', function() {
    $('.pane-anchor').trigger('arv:pane:reload');
});

$(document).on('ready ajax:complete', function() {
    // Panes marked arv-refresh-on-log-event should be refreshed
    $('.pane-anchor.arv-refresh-on-log-event').on('arv-log-event', function(e) {
        $(e.target).trigger('arv:pane:reload');
    });
});

// If there is a 'tab counts url' in the nav-tabs element then use it to get some javascript that will update them
$(document).on('ready count-change', function() {
    var tabCountsUrl = $('ul.nav-tabs').data('tab-counts-url');
    if( tabCountsUrl && tabCountsUrl.length ) {
        $.get( tabCountsUrl );
    }
});
