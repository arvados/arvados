// Load tab panes on demand. See app/views/application/_content.html.erb

// Fire when a tab is selected/clicked.
$(document).on('shown.bs.tab', '[data-toggle="tab"]', function(event) {
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

    $(event.target).trigger('arv:pane:reload');
});

// Ask a refreshable pane to reload via ajax.
// Target of this event is the anchoring element that manages the pane.
// Panes can be in one of four states: not loaded (no state classes), pane-loading, pane-loading+pane-stale, pane-loaded
$(document).on('arv:pane:reload', function(e) {
    e.stopPropagation();

    var etarget = $(e.target);

    if (etarget.hasClass('pane-loading')) {
        // Already loading, mark stale to schedule a reload after this one.
        //console.log(e.target.id + " stale");
        etarget.addClass('pane-stale');
        return;
    }

    if (etarget.hasClass('pane-no-auto-reload') && etarget.hasClass('pane-loaded')) {
        // Have to explicitly remove pane-loaded if we want it to reload.
        return;
    }

    var throttle = etarget.attr('data-load-throttle');
    if (!throttle) {
        throttle = 3000;
    }
    var now = (new Date()).getTime();
    var loaded_at = etarget.attr('data-loaded-at');
    if (loaded_at && (now - loaded_at) < throttle) {
        setTimeout(function() {
            etarget.trigger('arv:pane:reload');
        });
        return;
    }

    var $pane = $(etarget.attr('href'));
    if ($pane.hasClass('active')) {
        //console.log(e.target.id + " loading");

        etarget.removeClass('pane-loaded');
        etarget.removeClass('pane-stale');
        etarget.addClass('pane-loading');

        var content_url = etarget.attr('data-pane-content-url');
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
                etarget.removeClass('pane-loading');
                etarget.addClass('pane-loaded');
                etarget.attr('data-loaded-at', (new Date()).getTime());
                $pane.trigger('arv:pane:loaded');

                //console.log(e.target.id + " loaded");

                if (etarget.hasClass('pane-stale')) {
                    etarget.trigger('arv:pane:reload');
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
                $pane.html('<div><p>' +
                        '<a href="#" class="btn btn-primary tab_reload">' +
                        '<i class="fa fa-fw fa-refresh"></i> ' +
                        'Reload tab</a></p><iframe></iframe></div>');
                $('.tab_reload', $pane).click(function() {
                    $pane.html('<div class="spinner spinner-32px spinner-h-center"></div>');
                    etarget.trigger('arv:pane:reload');
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
                etarget.addClass('pane-loaded');
            });
    } else {
        //console.log(etarget.attr('href') + " is not active");

        // When the user selects e.target tab, show a spinner instead of
        // old content while loading.
        etarget.removeClass('pane-loading');
        etarget.removeClass('pane-loaded');
        etarget.removeClass('pane-stale');

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
