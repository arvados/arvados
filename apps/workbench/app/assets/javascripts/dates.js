$(document).on('ajax:complete arv:pane:loaded ready', function() {
    $('[data-utc-date]').each(function(i, elm) {
        var re = /(\d\d\d\d)-(\d\d)-(\d\d) (\d\d):(\d\d):(\d\d) UTC/;
        var v = $(elm).attr('data-utc-date').match(re);
        if (v) {
            var ts = new Date(Date.UTC(v[1], v[2]-1, v[3], v[4], v[5], v[6]));
            $(elm).text(ts.toLocaleTimeString() + " " + ts.toLocaleDateString());
        }
    });
});
