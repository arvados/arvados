function maybe_load_more_content() {
    var scroller = this;        // element with scroll bars
    var container;              // element that receives new content
    var src;                    // url for retrieving content
    var scrollHeight;
    var spinner, colspan;
    scrollHeight = scroller.scrollHeight || $('body')[0].scrollHeight;
    var num_scrollers = $(window).data("arv-num-scrollers");
    if ($(scroller).scrollTop() + $(scroller).height()
        >
        scrollHeight - 50)
    {
      for (var i = 0; i < num_scrollers; i++) {
        $container = $($(this).data('infinite-container'+i));
        src = $container.attr('data-infinite-content-href');
        if (!src || !$container.is(':visible'))
          continue;

        // Don't start another request until this one finishes
        $container.attr('data-infinite-content-href', null);
        spinner = '<div class="spinner spinner-32px spinner-h-center"></div>';
        if ($(container).is('table,tbody,thead,tfoot')) {
            // Hack to determine how many columns a new tr should have
            // in order to reach full width.
            colspan = $container.closest('table').
                find('tr').eq(0).find('td,th').length;
            if (colspan == 0)
                colspan = '*';
            spinner = ('<tr class="spinner"><td colspan="' + colspan + '">' +
                       spinner +
                       '</td></tr>');
        }
        $container.append(spinner);
        $.ajax(src,
               {dataType: 'json',
                type: 'GET',
                data: {},
                context: {container: $container, src: src}}).
            always(function() {
                $(this.container).find(".spinner").detach();
            }).
            fail(function(jqxhr, status, error) {
                if (jqxhr.readyState == 0 || jqxhr.status == 0) {
                    message = "Cancelled."
                } else if (jqxhr.responseJSON && jqxhr.responseJSON.errors) {
                    message = jqxhr.responseJSON.errors.join("; ");
                } else {
                    message = "Request failed.";
                }
                // TODO: report this to the user.
                console.log(message);
                $(this.container).attr('data-infinite-content-href', this.src);
            }).
            done(function(data, status, jqxhr) {
                $(this.container).append(data.content);
                $(this.container).attr('data-infinite-content-href', data.next_page_href);
            });
        break;
     }
   }
}

$(document).
    on('ready ajax:complete', function() {
        var num_scrollers = 0;
        $('[data-infinite-scroller]').each(function() {
            var $scroller = $($(this).attr('data-infinite-scroller'));
            if (!$scroller.hasClass('smart-scroll') &&
                'scroll' != $scroller.css('overflow-y'))
                $scroller = $(window);
            $scroller.
                addClass('infinite-scroller').
                data('infinite-container'+num_scrollers, this).
                on('scroll', maybe_load_more_content);
            num_scrollers++;
        });
        $(window).data("arv-num-scrollers", num_scrollers);
    });
