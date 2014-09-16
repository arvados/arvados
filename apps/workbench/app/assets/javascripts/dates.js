jQuery(function($){
$(document).on('ajax:complete arv:pane:loaded ready', function() {
    $('[data-utc-date]').each(function(i, elm) {
            console.log("woble!");
        var v = $(elm).attr('data-utc-date').match(/(\d\d\d\d)-(\d\d)-(\d\d) (\d\d):(\d\d):(\d\d) UTC/);
        if (!v) {
            v = $(elm).attr('data-utc-date').match(/(\d\d\d\d)-(\d\d)-(\d\d)T(\d\d):(\d\d):(\d\d)Z/);
        }

        if (v) {
            var ts = new Date(Date.UTC(v[1], v[2]-1, v[3], v[4], v[5], v[6]));
            if ($(elm).attr('data-utc-date-opts') && $(elm).attr('data-utc-date-opts').match(/noseconds/)) {
                $(elm).text((ts.getHours() > 12 ? (ts.getHours()-12) : ts.getHours())
                            + ":" + (ts.getMinutes() < 10 ? '0' : '') + ts.getMinutes()
                            + (ts.getHours() > 12 ? " PM " : " AM ")
                            + ts.toLocaleDateString());
            } else {
                $(elm).text(ts.toLocaleTimeString() + " " + ts.toLocaleDateString());
            }
        }
    });
});
});
