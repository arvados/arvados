$('div').remove('.request_shell_access_failed');
$('div').remove('.no_shell_access_msg');
$('div').remove('.shell_access_requested');
$('.no_shell_access').append('<div class="alert alert-success"><p class="contain-align-left">A request for shell access was sent.</p></div>');
var timestamp = new Date();
localStorage.setItem("request_shell_access", "A request for shell access was sent on " +
                                              timestamp.toLocaleDateString() +
                                              " at " + timestamp.toLocaleTimeString());
   
