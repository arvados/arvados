$(document).ready(function(){
    var $noShellAccessDiv = $('#no_shell_access');
    if ($noShellAccessDiv.length) {
      requestSent = localStorage.getItem('request_shell_access');
      if (requestSent != null) {
        $("#shell_access_requested_msg").html(requestSent)
      } else {
        $('div').remove('.shell_access_requested');
      }
    }
  });

$(document).
  on('click', "#request_shell_submit", function(e){
    $(this).html('Sending request ...');
    $(this).prop('disabled', true);
    $('div').remove('.request_shell_access_failed');

    $.ajax('/').
      success(function(data, status, jqxhr) {
        $('div').remove('.no_shell_access_msg');
        $('div').remove('.shell_access_requested');

        $('.no_shell_access').append('<div class="alert alert-success"><p class="contain-align-left">A request for shell access was sent.</p></div>');
        var timestamp = new Date();
        localStorage.setItem("request_shell_access", "A request for shell access was sent on " +
                                                      timestamp.toLocaleDateString() +
                                                      " at " + timestamp.toLocaleTimeString());
      }).
      fail(function(jqxhr, status, error) {
        var $sendButton = $('#request_shell_submit');
        $sendButton.html('Request shell access');
        $sendButton.prop('disabled', false);
        $('.no_shell_access').append('<div class="request_shell_access_failed alert alert-danger"><p class="contain-align-left">Something went wrong. Please try again.</p></div>');
      });
  });
