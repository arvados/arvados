$(document).on('ajax:error', function(e, xhr, status, error) {
    var errorMessage = '' + status + ': ' + error;
    // $btn is the element (button/link) that initiated the failed request.
    var $btn = $(e.target);
    // Populate some elements with the error text (e.g., a <p> in an alert div)
    $($btn.attr('data-on-error-write')).text(errorMessage);
    // Show some elements (e.g., an alert div)
    $($btn.attr('data-on-error-show')).show();
    // Hide some elements (e.g., a success/normal div)
    $($btn.attr('data-on-error-hide')).hide();
}).on('ajax:success', function(e) {
    var $btn = $(e.target);
    $($btn.attr('data-on-success-show')).show();
    $($btn.attr('data-on-success-hide')).hide();
});
