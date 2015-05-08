function maybe_load_more_content(event) {
    var scroller = this;
    var $container = $(event.data.container);
    var src;                     // url for retrieving content
    var scrollHeight;
    var spinner, colspan;
    var serial = Date.now();
    var params;
    scrollHeight = scroller.scrollHeight || $('body')[0].scrollHeight;
    if ($(scroller).scrollTop() + $(scroller).height()
        >
        scrollHeight - 50)
    {
        if (!$container.attr('data-infinite-content-href0')) {
            // Remember the first page source url, so we can refresh
            // from page 1 later.
            $container.attr('data-infinite-content-href0',
                            $container.attr('data-infinite-content-href'));
        }
        src = $container.attr('data-infinite-content-href');
        if (!src || !$container.is(':visible'))
            // Finished
            return;

        // Don't start another request until this one finishes
        $container.attr('data-infinite-content-href', null);
        spinner = '<div class="spinner spinner-32px spinner-h-center"></div>';
        if ($container.is('table,tbody,thead,tfoot')) {
            // Hack to determine how many columns a new tr should have
            // in order to reach full width.
            colspan = $container.closest('table').
                find('tr').eq(0).find('td,th').length;
            if (colspan == 0)
                colspan = '*';
            spinner = ('<tr class="spinner"><td colspan="' + colspan + '">' +
                       spinner +
                       '</td></tr>');
        }
        $container.find(".spinner").detach();
        $container.append(spinner);
        $container.data('data-infinite-serial', serial);

        if (src == $container.attr('data-infinite-content-href0')) {
            // If we're loading the first page, collect filters from
            // various sources.
            params = mergeInfiniteContentParams($container);
            $.each(params, function(k,v) {
                if (v instanceof Object) {
                    params[k] = JSON.stringify(v);
                }
            });
        } else {
            // If we're loading page >1, ignore other filtering
            // mechanisms and just use the "next page" URI from the
            // previous page's response. Aside from avoiding race
            // conditions (where page 2 could have different filters
            // than page 1), this allows the server to use filters in
            // the "next page" URI to achieve paging. (To apply any
            // new filters effectively, we need to load page 1 again
            // anyway.)
            params = {};
        }

        $.ajax(src,
               {dataType: 'json',
                type: 'GET',
                data: params,
                context: {container: $container, src: src, serial: serial}}).
            fail(function(jqxhr, status, error) {
                var $faildiv;
                var $container = this.container;
                if ($container.data('data-infinite-serial') != this.serial) {
                    // A newer request is already in progress.
                    return;
                }
                if (jqxhr.readyState == 0 || jqxhr.status == 0) {
                    return;   // User may have navigated away; skip.
                } else if (jqxhr.responseJSON && jqxhr.responseJSON.errors) {
                    message = jqxhr.responseJSON.errors.join("; ");
                } else {
                    message = "Request failed.";
                }
                // TODO: report the message to the user.
                console.log(message);
                $faildiv = $('<div />').
                    attr('data-infinite-content-href', this.src).
                    addClass('infinite-retry').
                    append('<span class="fa fa-warning" /> Oops, request failed. <button class="btn btn-xs btn-primary">Retry</button>');
                $container.find('div.spinner').replaceWith($faildiv);
            }).
            done(function(data, status, jqxhr) {
                if ($container.data('data-infinite-serial') != this.serial) {
                    // A newer request is already in progress.
                    return;
                }
                $container.find(".spinner").detach();
                $container.append(data.content);
                $container.attr('data-infinite-content-href', data.next_page_href);
            });
     }
}

function ping_all_scrollers() {
    // Send a scroll event to all scroll listeners that might need
    // updating. Adding infinite-scroller class to the window element
    // doesn't work, so we add it explicitly here.
    $('.infinite-scroller').add(window).trigger('scroll');
}

function mergeInfiniteContentParams($container) {
    var params = {};
    // Combine infiniteContentParams from multiple sources. This
    // mechanism allows each of several components to set and
    // update its own set of filters, without having to worry
    // about stomping on some other component's filters.
    //
    // For example, filterable.js writes filters in
    // infiniteContentParamsFilterable ("search for text foo")
    // without worrying about clobbering the filters set up by the
    // tab pane ("only show jobs and pipelines in this tab").
    $.each($container.data(), function(datakey, datavalue) {
        // Note: We attach these data to DOM elements using
        // <element data-foo-bar="baz">. We store/retrieve them
        // using $('element').data('foo-bar'), although
        // .data('fooBar') would also work. The "all data" hash
        // returned by $('element').data(), however, always has
        // keys like 'fooBar'. In other words, where we have a
        // choice, we stick with the 'foo-bar' style to be
        // consistent with HTML. Here, our only option is
        // 'fooBar'.
        if (/^infiniteContentParams/.exec(datakey)) {
            if (datavalue instanceof Object) {
                $.each(datavalue, function(hkey, hvalue) {
                    if (hvalue instanceof Array) {
                        params[hkey] = (params[hkey] || []).
                            concat(hvalue);
                    } else if (hvalue instanceof Object) {
                        $.extend(params[hkey], hvalue);
                    } else {
                        params[hkey] = hvalue;
                    }
                });
            }
        }
    });
    return params;
}

function setColumnSort( $container, $header, direction ) {
    // $container should be the tbody or whatever has all the infinite table data attributes
    // $header should be the th with a preset data-sort-order attribute
    // direction should be "asc" or "desc"
    // This function returns the order by clause for this column header as a string

    // First reset all sort directions
    $('th[data-sort-order]').removeData('sort-order-direction');
    // set the current one
    $header.data('sort-order-direction', direction);
    // change the ordering parameter
    var paramsAttr = 'infinite-content-params-' + $container.data('infinite-content-params-attr');
    var params = $container.data(paramsAttr) || {};
    params.order = $header.data('sort-order').split(",").join( ' ' + direction + ', ' ) + ' ' + direction;
    $container.data(paramsAttr, params);
    // show the correct icon next to the column header
    $container.trigger('sort-icons');

    return params.order;
}

$(document).
    on('click', 'div.infinite-retry button', function() {
        var $retry_div = $(this).closest('.infinite-retry');
        var $container = $(this).closest('.infinite-scroller-ready')
        $container.attr('data-infinite-content-href',
                        $retry_div.attr('data-infinite-content-href'));
        $retry_div.
            replaceWith('<div class="spinner spinner-32px spinner-h-center" />');
        ping_all_scrollers();
    }).
    on('refresh-content', '[data-infinite-scroller]', function() {
        // Clear all rows, reset source href to initial state, and
        // (if the container is visible) start loading content.
        var first_page_href = $(this).attr('data-infinite-content-href0');
        if (!first_page_href)
            first_page_href = $(this).attr('data-infinite-content-href');
        $(this).
            html('').
            attr('data-infinite-content-href', first_page_href);
        ping_all_scrollers();
    }).
    on('ready ajax:complete', function() {
        $('[data-infinite-scroller]').each(function() {
            if ($(this).hasClass('infinite-scroller-ready'))
                return;
            $(this).addClass('infinite-scroller-ready');

            // deal with sorting if there is any, and if it was set on this page for this tab already
            if( $('th[data-sort-order]').length ) {
                var tabId = $(this).closest('div.tab-pane').attr('id');
                if( hasHTML5History() && history.state !== undefined && history.state !== null && history.state.order !== undefined && history.state.order[tabId] !== undefined ) {
                    // we will use the list of one or more table columns associated with this header to find the right element
                    // see sortable_columns as it is passed to render_pane in the various tab .erbs (e.g. _show_jobs_and_pipelines.html.erb)
                    var strippedColumns = history.state.order[tabId].replace(/\s|\basc\b|\bdesc\b/g,'');
                    var sortDirection = history.state.order[tabId].split(" ")[1].replace(/,/,'');
                    $columnHeader = $(this).closest('table').find('[data-sort-order="'+ strippedColumns +'"]');
                    setColumnSort( $(this), $columnHeader, sortDirection );
                } else {
                    // otherwise just reset the sort icons
                    $(this).trigger('sort-icons');
                }
            }

            // $scroller is the DOM element that hears "scroll"
            // events: sometimes it's a div, sometimes it's
            // window. Here, "this" is the DOM element containing the
            // result rows. We pass it to maybe_load_more_content in
            // event.data.
            var $scroller = $($(this).attr('data-infinite-scroller'));
            if (!$scroller.hasClass('smart-scroll') &&
                'scroll' != $scroller.css('overflow-y'))
                $scroller = $(window);
            $scroller.
                addClass('infinite-scroller').
                on('scroll resize', { container: this }, maybe_load_more_content).
                trigger('scroll');
        });
    }).
    on('shown.bs.tab', 'a[data-toggle="tab"]', function(event) {
        $(event.target.getAttribute('href') + ' [data-infinite-scroller]').
            trigger('scroll');
    }).
    on('click', 'th[data-sort-order]', function() {
        var direction = $(this).data('sort-order-direction');
        // reverse the current direction, or do ascending if none
        if( direction === undefined || direction === 'desc' ) {
            direction = 'asc';
        } else {
            direction = 'desc';
        }

        var $container = $(this).closest('table').find('[data-infinite-content-params-attr]');

        var order = setColumnSort( $container, $(this), direction );

        // put it in the browser history state if browser allows it
        if( hasHTML5History() ) {
            var tabId = $(this).closest('div.tab-pane').attr('id');
            var state =  history.state || {};
            if( state.order === undefined ) {
                state.order = {};
            }
            state.order[tabId] = order;
            history.replaceState( state, null, null );
        }

        $container.trigger('refresh-content');
    }).
    on('sort-icons', function() {
        // set or reset the icon next to each sortable column header according to the current direction attribute
        $('th[data-sort-order]').each(function() {
            $(this).find('i').remove();
            var direction = $(this).data('sort-order-direction');
            if( direction !== undefined ) {
                $(this).append('<i class="fa fa-sort-' + direction + '"/>');
            } else {
                $(this).append('<i class="fa fa-sort"/>');
            }
        });
    });
