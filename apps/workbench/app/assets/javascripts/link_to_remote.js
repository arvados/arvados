$.rails.href = function(element) {
    if (element.is('a')) {
        // data-remote=true links must put their remote targets in
        // data-remote-href="..." instead of href="...".  This helps
        // us avoid accidentally using the same href="..." in both the
        // remote (Rails UJS) and non-remote (native browser) handlers
        // -- which differ greatly in how they use that value -- and
        // forgetting to test any non-remote cases like "open in new
        // tab". If you really want copy-link-address/open-in-new-tab
        // to work on a data-remote=true link, supply the
        // copy-and-pastable URI in href in addition to the AJAX URI
        // in data-remote-href.
        //
        // (Currently, the only places we make any remote links are
        // link_to() in ApplicationHelper, which renames href="..." to
        // data-remote-href="...", and select_modal, which builds a
        // data-remote=true link on the client side.)
        return element.data('remote-href');
    } else {
        // Normal rails-ujs behavior.
        return element.attr('href');
    }
}
