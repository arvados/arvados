$(document).
    on('ready ajax:complete', function() {
        $("[data-toggle='x-editable']").click(function(e) {
            e.stopPropagation();
            $($(this).attr('data-toggle-selector')).editable('toggle');
        });
    }).on('paste keyup change', 'input.search-folder-contents', function() {
        var q = new RegExp($(this).val(), 'i');
        $(this).closest('div.panel').find('tbody tr').each(function() {
            $(this).toggle(!!$(this).text().match(q));
        });
    });
