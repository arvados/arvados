function maybe_load_more_content() {
    var scroller = this;        // element with scroll bars
    var container;              // element that receives new content
    var src;                    // url for retrieving content
    var scrollHeight;
    var spinner, colspan;
    scrollHeight = scroller.scrollHeight || $('body')[0].scrollHeight;
    if ($(scroller).scrollTop() + $(scroller).height()
        >
        scrollHeight - 50) {
        container = $(this).data('infinite-container');
        src = $(container).attr('data-infinite-content-href');
        if (!src)
            // Finished
            return;
        // Don't start another request until this one finishes
        $(container).attr('data-infinite-content-href', null);
        spinner = '<div class="spinner spinner-32px spinner-h-center"></div>';
        if ($(container).is('table,tbody,thead,tfoot')) {
            // Hack to determine how many columns a new tr should have
            // in order to reach full width.
            colspan = $(container).closest('table').
                find('tr').eq(0).find('td,th').length;
            if (colspan == 0)
                colspan = '*';
            spinner = ('<tr class="spinner"><td colspan="' + colspan + '">' +
                       spinner +
                       '</td></tr>');
        }
        $(container).find(".spinner").detach();
        $(container).append(spinner);
        $.ajax(src,
               {dataType: 'json',
                type: 'GET',
                data: {},
                context: {container: container, src: src}}).
            always(function() {
            }).
            fail(function(jqxhr, status, error) {
                var $faildiv;
                if (jqxhr.readyState == 0 || jqxhr.status == 0) {
                    message = "Cancelled."
                } else if (jqxhr.responseJSON && jqxhr.responseJSON.errors) {
                    message = jqxhr.responseJSON.errors.join("; ");
                } else {
                    message = "Request failed.";
                }
                // TODO: report the message to the user.
                console.log(message);
                $faildiv = $('<div />').
                    attr('data-infinite-content-href', this.src).
                    addClass('infinite-retry').
                    append('<span class="fa fa-warning" /> Oops, request failed. <button class="btn btn-xs btn-primary">Retry</button>');
                $(this.container).find('div.spinner').replaceWith($faildiv);
            }).
            done(function(data, status, jqxhr) {
                $(this.container).find(".spinner").detach();
                $(this.container).append(data.content);
                $(this.container).attr('data-infinite-content-href', data.next_page_href);
            });
    }
}
$(document).
    on('click', 'div.infinite-retry button', function() {
        var $retry_div = $(this).closest('.infinite-retry');
        var $scroller = $(this).closest('.infinite-scroller')
        $scroller.attr('data-infinite-content-href',
                       $retry_div.attr('data-infinite-content-href'));
        $retry_div.replaceWith('<div class="spinner spinner-32px spinner-h-center" />');
        $scroller.trigger('scroll');
    }).
    on('ready ajax:complete', function() {
        $('[data-infinite-scroller]').each(function() {
            var $scroller = $($(this).attr('data-infinite-scroller'));
            if (!$scroller.hasClass('smart-scroll') &&
                'scroll' != $scroller.css('overflow-y'))
                $scroller = $(window);
            $scroller.
                addClass('infinite-scroller').
                data('infinite-container', this).
                on('scroll', maybe_load_more_content);
        });
    });
