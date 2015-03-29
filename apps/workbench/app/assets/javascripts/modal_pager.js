// Usage:
//
// 1. Add some buttons to your modal, one with class="pager-next" and
// one with class="pager-prev".
//
// 2. Put multiple .modal-body sections in your modal.
//
// 3. Add a "pager-count" div where page count is shown.
// For ex: "1 of 10" when showing first page of 10 pages.

$(document).on('click', '.modal .pager-next', function() {
    var $modal = $(this).parents('.modal');
    $modal.data('page', ($modal.data('page') || 0) + 1).trigger('pager:render');
    return false;
}).on('click', '.modal .pager-prev', function() {
    var $modal = $(this).parents('.modal');
    $modal.data('page', ($modal.data('page') || 1) - 1).trigger('pager:render');
    return false;
}).on('ready ajax:success', function() {
    $('.modal').trigger('pager:render');
}).on('pager:render', '.modal', function() {
    var $modal = $(this);
    var page = $modal.data('page') || 0;
    var $panes = $('.modal-body', $modal);
    if (page >= $panes.length) {
        // Somehow moved past end
        page = $panes.length - 1;
        $modal.data('page', page);
    } else if (page < 0) {
        page = 0;
    }

    var $pager_count = $('.pager-count', $modal);
    $pager_count.text((page+1) + " of " + $panes.length);

    var selected = $panes.hide().eq(page).show();
    enableButton($('.pager-prev', $modal), page > 0);
    enableButton($('.pager-next', $modal), page < $panes.length - 1);
    function enableButton(btn, ok) {
        btn.prop('disabled', !ok).
            toggleClass('btn-primary', ok).
            toggleClass('btn-default', !ok);
    }
});
