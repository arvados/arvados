$(document).
    on('paste keyup input', 'input[type=text].filterable-control', function() {
        var infinite_container = $(this).closest('[data-infinite-scroller]');
        var q = new RegExp($(this).val(), 'i');
        var $target = $($(this).attr('data-filterable-target'));
        $target.
            addClass('filterable-container').
            data('q', q).
            trigger('refresh');
        if ($target.is('[data-infinite-scroller]')) {
            params = $target.data('infinite-content-params') || {};
            params.filters = JSON.stringify([['any', 'like', '%' + $(this).val() + '%']]);
            $target.data('infinite-content-params', params);
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
