$(document).
  on('click', "#request_shell_submit", function(e){
    $(this).html('Sending request ...');
    $(this).prop('disabled', true);
    $('div').remove('.request_shell_access_failed');

    $.ajax('/').
      success(function(data, status, jqxhr) {
        $('div').remove('.no_shell_access_msg');
        $('.no_shell_access').append('<div class="alert alert-success"><p class="contain-align-left">Request sent for shell access.</p></div>');
        localStorage.setItem("request_shell_access", "sent");
      }).
      fail(function(jqxhr, status, error) {
        var $sendButton = $('#request_shell_submit');
        $sendButton.html('Request shell access');
        $sendButton.prop('disabled', false);
        $('.no_shell_access').append('<div class="request_shell_access_failed alert alert-danger"><p class="contain-align-left">Something went wrong. Please try again.</p></div>');
      });
  });
