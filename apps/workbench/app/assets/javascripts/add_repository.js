$(document).on('shown.bs.modal', '#add-repository-modal', function(event) {
    $('input[type=text]', event.target).val('');
    $('#add-repository-error', event.target).hide();
}).on('submit', '#add-repository-form', function(event) {
    var $form = $(event.target),
    $submit = $(':submit', $form),
    $error = $('#add-repository-error', $form),
    repo_owner_uuid = $('input[name="add_repo_owner_uuid"]', $form).val(),
    repo_prefix = $('input[name="add_repo_prefix"]', $form).val(),
    repo_basename = $('input[name="add_repo_basename"]', $form).val();

    $submit.prop('disabled', true);
    $error.hide();
    $.ajax('/repositories',
           {method: 'POST',
            dataType: 'json',
            data: {repository: {owner_uuid: repo_owner_uuid,
                                name: repo_prefix + repo_basename}},
            context: $form}).
        done(function(data, status, jqxhr) {
            location.reload();
        }).
        fail(function(jqxhr, status, error) {
            var errlist = jqxhr.responseJSON.errors;
            var errmsg;
            if (Array.isArray(errlist)) {
                errmsg = errlist.join();
            } else {
                errmsg = ("The server returned an error when making " +
                          "this repository (status " + jqxhr.status +
                          ": " + errlist + ").");
            }
            $error.text(errmsg);
            $error.show();
            $submit.prop('disabled', false);
        });
    return false;
});
