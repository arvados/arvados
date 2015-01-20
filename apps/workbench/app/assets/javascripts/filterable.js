// filterable.js shows/hides content when the user operates
// search/select widgets. For "infinite scroll" content, it passes the
// filters to the server and retrieves new content. For other content,
// it filters the existing DOM elements using jQuery show/hide.
//
// Usage:
//
// 1. Add the "filterable" class to each filterable content item.
// Typically, each item is a 'tr' or a 'div class="row"'.
//
// <div id="results">
//   <div class="filterable row">First row</div>
//   <div class="filterable row">Second row</div>
// </div>
//
// 2. Add the "filterable-control" class to each search/select widget.
// Also add a data-filterable-target attribute with a jQuery selector
// for an ancestor of the filterable items, i.e., the container in
// which this widget should apply filtering.
//
// <input class="filterable-control" data-filterable-target="#results"
//        type="text" />
//
// Supported widgets:
//
// <input type="text" ... />
//
// The input value is used as a regular expression. Rows with content
// matching the regular expression are shown.
//
// <select ... data-filterable-attribute="data-example-attr">
//  <option value="foo">Foo</option>
//  <option value="">Show all</option>
// </select>
//
// When the user selects the "Foo" option, rows with
// data-example-attr="foo" are shown, and all others are hidden. When
// the user selects the "Show all" option, all rows are shown.
//
// Notes:
//
// When multiple filterable-control widgets operate on the same
// data-filterable-target, items must pass _all_ filters in order to
// be shown.
//
// If one data-filterable-target is the parent of another
// data-filterable-target, results are undefined. Don't do this.
//
// Combining "select" filterable-controls with infinite-scroll is not
// yet supported.

function updateFilterableQueryNow($target) {
    var newquery = $target.data('filterable-query-new');
    var params = $target.data('infinite-content-params-filterable') || {};
    params.filters = [['any', 'ilike', '%' + newquery + '%']];
    $target.data('infinite-content-params-filterable', params);
    $target.data('filterable-query', newquery);
}

$(document).
    on('ready ajax:success', function() {
        // Copy any initial input values into
        // data-filterable-query[-new].
        $('input[type=text].filterable-control').each(function() {
            var $this = $(this);
            var $target = $($this.attr('data-filterable-target'));
            if ($target.data('filterable-query-new') === undefined) {
                $target.data('filterable-query', $this.val());
                $target.data('filterable-query-new', $this.val());
                updateFilterableQueryNow($target);
            }
        });
        $('[data-infinite-scroller]').on('refresh-content', '[data-filterable-query]', function(e) {
            // If some other event causes a refresh-content event while there
            // is a new query waiting to cooloff, we should use the new query
            // right away -- otherwise we'd launch an extra ajax request that
            // would have to be reloaded as soon as the cooloff period ends.
            if (this != e.target)
                return;
            if ($(this).data('filterable-query') == $(this).data('filterable-query-new'))
                return;
            updateFilterableQueryNow($(this));
        });
    }).
    on('paste keyup input', 'input[type=text].filterable-control', function(e) {
        var regexp;
        if (this != e.target) return;
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
                updateFilterableQueryNow($target);
                $target.trigger('refresh-content');
            }, 250));
        } else {
            // Target does not have infinite-scroll capability. Just
            // filter the rows in the browser using a RegExp.
            regexp = undefined;
            try {
                regexp = new RegExp($(this).val(), 'i');
            } catch(e) {
                if (e instanceof SyntaxError) {
                    // Invalid/partial regexp. See 'has-error' below.
                } else {
                    throw e;
                }
            }
            $target.
                toggleClass('has-error', regexp === undefined).
                addClass('filterable-container').
                data('q', regexp).
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
