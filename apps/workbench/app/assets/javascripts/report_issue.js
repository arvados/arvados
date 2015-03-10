$(document).
  on('click', "#report-issue-submit", function(e){
    $(this).html('Sending');
    $(this).prop('disabled', true);
    var $cancelButton = $('#report-issue-cancel');
    if ($cancelButton) {
      $cancelButton.html('Close');
    }
    $('div').remove('.modal-footer-status');

    $.ajax('/report_issue', {
        type: 'POST',
        data: $(this).parents('form').serialize()
    }).success(function(data, status, jqxhr) {
        var $sendButton = $('#report-issue-submit');
        $sendButton.html('Report sent');
        $('div').remove('.modal-footer-status');
        $('.modal-footer').append('<div><br/></div><div class="modal-footer-status alert alert-success"><p class="contain-align-left">Thanks for reporting this issue!</p></div>');
    }).fail(function(jqxhr, status, error) {
        var $sendButton = $('#report-issue-submit');
        if ($sendButton && $sendButton.prop('disabled')) {
          $('div').remove('.modal-footer-status');
          $('.modal-footer').append('<div><br/></div><div class="modal-footer-status alert alert-danger"><p class="contain-align-left">We are sorry. We could not submit your report! We really want this to work, though -- please try again.</p></div>');
          $sendButton.html('Send problem report');
          $sendButton.prop('disabled', false);
        }
        var $cancelButton = $('#report-issue-cancel');
        $cancelButton.html('Cancel');
    });
    return false;
  });
