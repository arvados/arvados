// ajax handling for report-issue function
$('#report-issue-modal-window form').
  on('ajax:send', function() {
    var $sendButton = $('#report-issue-submit');
    if ($sendButton) {
      $sendButton.html('Sending...');
      $sendButton.attr('disabled',true);
    }
    var $cancelButton = $('#report-issue-cancel');
    if ($cancelButton) {
      $cancelButton.html('Close');
    }
    $('div').remove('.modal-footer-status');
  }).
  on('ajax:success', function() {
    var $sendButton = $('#report-issue-submit');
    if ($sendButton  && $sendButton.prop('disabled')) {
      $sendButton.html('Report sent');
      $('div').remove('.modal-footer-status');
      $('.modal-footer').append('<div class="modal-footer-status alert alert-success"><div><p align="left">Thanks for reporting this issue!</p></div></div>');
    }
  }).
  on('ajax:failure', function() {
    var $sendButton = $('#report-issue-submit');
    if ($sendButton && $sendButton.prop('disabled')) {
      $('div').remove('.modal-footer-status');
      $('.modal-footer').append('<div class="modal-footer-status alert alert-danger"></br><p align="left">We are sorry. We could not submit your report! We really want this to work, though -- please try again.</p></div>');
      $sendButton.html('Send problem report');
      $sendButton.attr('disabled',false);
    }
    var $cancelButton = $('#report-issue-cancel');
    if ($cancelButton) {
      var text = document.getElementById('report-issue-cancel').firstChild;
      $cancelButton.html('Cancel');
    }
  });
