$(document).
    on('paste keyup change', 'input.search-folder-contents', function() {
        var q = new RegExp($(this).val(), 'i');
        $(this).closest('div.panel').find('tbody tr').each(function() {
            $(this).toggle(!!$(this).text().match(q));
        });
    }).on('change', 'select[data-filter-rows-by]', function() {
        var val = $(this).val();
        var filterby = $(this).attr('data-filter-rows-by');
        var $target = $($(this).attr('data-filter-in'));
        if (val == '') {
            $target.find('.filterable').show();
        } else {
            $target.find('.filterable').hide();
            console.log('.filterable[' + filterby + '="' + val + '"]');
            $.each(val.split(" "), function(i, e) {
                $target.find('.filterable['+filterby+'="'+e+'"]').show();
            });
        }
    });
