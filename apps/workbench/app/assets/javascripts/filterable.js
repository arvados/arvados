$(document).
    on('paste keyup change', 'input[type=text].filterable-control', function() {
        var q = new RegExp($(this).val(), 'i');
        $($(this).attr('data-filterable-target')).
            addClass('filterable-container').
            data('q', q).
            trigger('refresh');
    }).on('refresh', '.filterable-container', function() {
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
        $('.infinite-scroller').trigger('scroll');
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
        $('.filterable-control').trigger('change');
    });
