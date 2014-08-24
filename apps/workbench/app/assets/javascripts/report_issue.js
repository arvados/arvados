$(document).
  on('click', "#report-issue-submit", function(e){
    $(this).html('Sending');
    $(this).attr('disabled', true);
    var $cancelButton = $('#report-issue-cancel');
    if ($cancelButton) {
      $cancelButton.html('Close');
    }
    $('div').remove('.modal-footer-status');

    $.ajax().
      success(function(data, status, jqxhr) {
        var $sendButton = $('#report-issue-submit');
        $sendButton.html('Report sent');
        $('div').remove('.modal-footer-status');
        $('.modal-footer').append('<div class="modal-footer-status alert alert-success"><div><p align="left">Thanks for reporting this issue!</p></div></div>');
      }).
      fail(function(jqxhr, status, error) {
        var $sendButton = $('#report-issue-submit');
        if ($sendButton && $sendButton.prop('disabled')) {
        $('div').remove('.modal-footer-status');
        $('.modal-footer').append('<div class="modal-footer-status alert alert-danger"></br><p align="left">We are sorry. We could not submit your report! We really want this to work, though -- please try again.</p></div>');
        $sendButton.html('Send problem report');
        $sendButton.attr('disabled',false);
      }
        var $cancelButton = $('#report-issue-cancel');
        var text = document.getElementById('report-issue-cancel').firstChild;
        $cancelButton.html('Cancel');
      }).
      always(function(data, status, jqxhr) {
      });

  });
