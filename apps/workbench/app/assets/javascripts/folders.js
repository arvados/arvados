$(document).
    on('ready ajax:complete', function() {
        $("[data-toggle='x-editable']").click(function(e) {
            e.stopPropagation();
            $($(this).attr('data-toggle-selector')).editable('toggle');
        });
    });
