// Load tab panes on demand. See app/views/application/_content.html.erb

// Fire when a tab is selected/clicked.
$(document).on('shown.bs.tab', '[data-toggle="tab"]', function(event) {
    // reload the pane (unless it's already loaded)
    $($(event.target).attr('href')).
        not('.pane-loaded').
        trigger('arv:pane:reload');
});

// Ask a refreshable pane to reload via ajax.
//
// Target of this event is the anchor element that manages the pane.  A reload
// consists of an AJAX call to load the "data-pane-content-url" and replace the
// contents of the DOM node pointed to by "href".
//
// There are four CSS classes set on the object to indicate its state:
// pane-loading, pane-stale, pane-loaded, pane-reload-pending
//
// There are five states based on the presence or absence of css classes:
//
// 1. no pane-* states means the pane must be loaded when the pane becomes active
//
// 2. "pane-loading" means an AJAX call has been made to reload the pane and we are
// waiting on a result
//
// 3. "pane-loading pane-stale" indicates a pane that is already loading has
// been invalidated and should schedule a reload immediately when the current
// load completes.  (This happens when there is a cluster of events, where the
// reload is triggered by the first event, but we want ensure that we
// eventually load the final quiescent state).
//
// 4. "pane-loaded" means the pane is up to date
//
// 5. "pane-loaded pane-reload-pending" indicates a reload is scheduled (but has
// not started yet), suppressing scheduling of any further reloads.
//
$(document).on('arv:pane:reload', '[data-pane-content-url]', function(e) {
    // $pane, the event target, is an element whose content is to be
    // replaced. Pseudoclasses on $pane (pane-loading, etc) encode the
    // current loading state.
    var $pane = $(e.target);

    var content_url = $pane.attr('data-pane-content-url');
    if (!content_url) {
        // When reloadable elements are nested, we can receive
        // arv:pane:reload events even though the selector in .on()
        // does not match e.target. Ignore such events.
        return;
    }

    e.stopPropagation();

    if ($pane.hasClass('pane-loading')) {
        // Already loading, mark stale to schedule a reload after this one.
        $pane.addClass('pane-stale');
        return;
    }

    // The default throttle (mininum milliseconds between refreshes)
    // can be overridden by an .arv-log-refresh-control element inside
    // the pane -- or, failing that, the pane element itself -- with a
    // data-load-throttle attribute. This allows the server to adjust
    // the throttle depending on the pane content.
    var throttle =
        $pane.find('.arv-log-refresh-control').attr('data-load-throttle') ||
        $pane.attr('data-load-throttle') ||
        15000;
    var now = (new Date()).getTime();
    var loaded_at = $pane.attr('data-loaded-at');
    var since_last_load = now - loaded_at;
    if (loaded_at && (since_last_load < throttle)) {
        if (!$pane.hasClass('pane-reload-pending')) {
            $pane.addClass('pane-reload-pending');
            setTimeout((function() {
                $pane.trigger('arv:pane:reload');
            }), throttle - since_last_load);
        }
        return;
    }

    // We know this doesn't have 'pane-loading' because we tested for it above
    $pane.removeClass('pane-reload-pending');
    $pane.removeClass('pane-loaded');
    $pane.removeClass('pane-stale');

    if (!$pane.hasClass('active') &&
        $pane.parent().hasClass('tab-content')) {
        // $pane is one of the content areas in a bootstrap tabs
        // widget, and it isn't the currently selected tab. If and
        // when the user does select the corresponding tab, it will
        // get a shown.bs.tab event, which will invoke this reload
        // function again (see handler above). For now, we just insert
        // a spinner, which will be displayed while the new content is
        // loading.
        $pane.html('<div class="spinner spinner-32px spinner-h-center"></div>');
        return;
    }

    $pane.addClass('pane-loading');

    $.ajax(content_url, {dataType: 'html', type: 'GET', context: $pane}).
        done(function(data, status, jqxhr) {
            // Preserve collapsed state
            var $pane = this;
            var collapsable = {};
            $(".collapse", this).each(function(i, c) {
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
            $pane.removeClass('pane-loading');
            $pane.addClass('pane-loaded');
            $pane.attr('data-loaded-at', (new Date()).getTime());
            $pane.trigger('arv:pane:loaded');

            if ($pane.hasClass('pane-stale')) {
                $pane.trigger('arv:pane:reload');
            }
        }).fail(function(jqxhr, status, error) {
            var $pane = this;
            var errhtml;
            var contentType = jqxhr.getResponseHeader('Content-Type');
            if (contentType && contentType.match(/\btext\/html\b/)) {
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
                      'Reload tab</a></p><iframe style="width: 100%"></iframe></div>');
            $('.tab_reload', $pane).click(function() {
                $(this).
                    html('<div class="spinner spinner-32px spinner-h-center"></div>').
                    closest('.pane-loaded').
                    attr('data-loaded-at', 0).
                    trigger('arv:pane:reload');
            });
            // We want to render the error in an iframe, in order to
            // avoid conflicts with the main page's element ids, etc.
            // In order to do that dynamically, we have to set a
            // timeout on the iframe window to load our HTML *after*
            // the default source (e.g., about:blank) has loaded.
            var iframe = $('iframe', $pane)[0];
            iframe.contentWindow.setTimeout(function() {
                $('body', iframe.contentDocument).html(errhtml);
                iframe.height = iframe.contentDocument.body.scrollHeight + "px";
            }, 1);
            $pane.removeClass('pane-loading');
            $pane.addClass('pane-loaded');
        });
});

// Mark all panes as stale/dirty. Refresh any 'active' panes.
$(document).on('arv:pane:reload:all', function() {
    $('[data-pane-content-url]').trigger('arv:pane:reload');
});

$(document).on('arv-log-event', '.arv-refresh-on-log-event', function(e) {
    // Panes marked arv-refresh-on-log-event should be refreshed
    $(e.target).trigger('arv:pane:reload');
});

// If there is a 'tab counts url' in the nav-tabs element then use it to get some javascript that will update them
$(document).on('ready count-change', function() {
    var tabCountsUrl = $('ul.nav-tabs').data('tab-counts-url');
    if( tabCountsUrl && tabCountsUrl.length ) {
        $.get( tabCountsUrl );
    }
});
