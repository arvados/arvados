$('#open_user_agreement input[name="checked[]"]').on('click', function() {
    var dialog = $('#open_user_agreement')[0]
    $('input[type=submit]', dialog).prop('disabled',false);
    $('input[name="checked[]"]', dialog).each(function(){
        if(!this.checked) {
            $('input[type=submit]', dialog).prop('disabled',true);
        }
    });
});
