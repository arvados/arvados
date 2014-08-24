$(document).
    on('paste keyup input', 'input[type=text].filterable-control', function() {
        var $target = $($(this).attr('data-filterable-target'));
        var currentquery = $target.data('filterable-query');
        if (currentquery === undefined) currentquery = '';
        if ($target.is('[data-infinite-scroller]')) {
            // We already know how to load content dynamically, so we
            // can do all filtering on the server side.

            if ($target.data('infinite-cooloff-timer') > 0) {
                // Clear a stale refresh-after-delay timer.
                clearTimeout($target.data('infinite-cooloff-timer'));
            }
            // Stash the new query string in the filterable container.
            $target.data('filterable-query-new', $(this).val());
            if (currentquery == $(this).val()) {
                // Don't mess with existing results or queries in
                // progress.
                return;
            }
            $target.data('infinite-cooloff-timer', setTimeout(function() {
                // If the user doesn't do any query-changing actions
                // in the next 1/4 second (like type or erase
                // characters in the search box), hide the stale
                // content and ask the server for new results.
                var newquery = $target.data('filterable-query-new');
                var params = $target.data('infinite-content-params') || {};
                params.filters = JSON.stringify([['any', 'ilike', '%' + newquery + '%']]);
                $target.data('infinite-content-params', params);
                $target.data('filterable-query', newquery);
                $target.trigger('refresh-content');
            }, 250));
        } else {
            // Target does not have infinite-scroll capability. Just
            // filter the rows in the browser using a RegExp.
            $target.
                addClass('filterable-container').
                data('q', new RegExp($(this).val(), 'i')).
                trigger('refresh');
        }
    }).on('refresh', '.filterable-container', function() {
        var $container = $(this);
        var q = $(this).data('q');
        var filters = $(this).data('filters');
        $('.filterable', this).hide().filter(function() {
            var $row = $(this);
            var pass = true;
            if (q && !$row.text().match(q))
                return false;
            if (filters) {
                $.each(filters, function(filterby, val) {
                    if (!val) return;
                    if (!pass) return;
                    pass = false;
                    $.each(val.split(" "), function(i, e) {
                        if ($row.attr(filterby) == e)
                            pass = true;
                    });
                });
            }
            return pass;
        }).show();

        // Show/hide each section heading depending on whether any
        // content rows are visible in that section.
        $('.row[data-section-heading]', this).each(function(){
            $(this).toggle($('.row.filterable[data-section-name="' +
                             $(this).attr('data-section-name') +
                             '"]:visible').length > 0);
        });

        // Load more content if the last result is showing.
        $('.infinite-scroller').add(window).trigger('scroll');
    }).on('change', 'select.filterable-control', function() {
        var val = $(this).val();
        var filterby = $(this).attr('data-filterable-attribute');
        var $target = $($(this).attr('data-filterable-target')).
            addClass('filterable-container');
        var filters = $target.data('filters') || {};
        filters[filterby] = val;
        $target.
            data('filters', filters).
            trigger('refresh');
    }).on('ajax:complete', function() {
        $('.filterable-control').trigger('input');
    });
