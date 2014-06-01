$(document).
    on('paste keyup change', 'input.search-folder-contents', function() {
        var q = new RegExp($(this).val(), 'i');
        $($(this).attr('data-search-target')).find('tbody').
            data('q', q).
            trigger('refresh');
    }).on('refresh', 'tbody', function() {
        var q = $(this).data('q');
        var filters = $(this).data('filters');
        $('tr', this).hide();
        $('tr', this).filter(function() {
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
        $('.infinite-scroller').trigger('scroll');
    }).on('change', 'select[data-filter-rows-by]', function() {
        var val = $(this).val();
        var filterby = $(this).attr('data-filter-rows-by');
        var $target = $($(this).attr('data-filter-target'));
        var filters = $target.data('filters') || {};
        filters[filterby] = val;
        $target.
            data('filters', filters).
            trigger('refresh');
    }).on('ajax:complete', function() {
        $('input.search-folder-contents').trigger('change');
        $('select[data-filter-rows-by]').trigger('change');
    });
