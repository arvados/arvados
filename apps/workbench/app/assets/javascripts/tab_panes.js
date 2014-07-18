$(document).on('shown.bs.tab', '[data-toggle="tab"]', function(e) {
    var content_url = $(e.target).attr('data-pane-content-url');
    var $pane = $($(e.target).attr('href'));
    if ($pane.hasClass('loaded'))
        return;
    $.ajax(content_url, {dataType: 'html', type: 'GET', context: $pane}).
        done(function(data, status, jqxhr) {
            $('> div > div', this).html(data);
            $(this).addClass('loaded');
        });
});

$(document).on('arv-log-event', function() {
    $('[data-pane-content-url]').removeClass('loaded');
    $('.tab-pane.active').trigger('shown.bs.tab');
});
