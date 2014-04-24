jQuery(function($){
    $(document).on('change', '.toggle-persist input[name=wants]', function() {
        var toggle_group = $(this).parents('[data-remote-href]').first();
        if (toggle_group.attr('data-persistent-state') == $(this).val()) {
            // When the user clicks the already-selected choice, or
            // the fail() handler below reverts state to the existing
            // state, don't start an AJAX request.
            return;
        }
        $.ajax(toggle_group.attr('data-remote-href'),
               {dataType: 'json',
                type: 'POST',
                data: {
                    value: $(this).val()
                },
                context: {
                    toggle_group: toggle_group,
                    input: this
                }
               }).
            done(function(data, status, jqxhr) {
                var context = this;
                $(document).trigger('ajax:complete');
                // Remove "danger" status in case a previous action failed
                $('label.btn-danger', context.toggle_group).
                    addClass('btn-info').
                    removeClass('btn-danger');
                // Update last-saved-state
                context.toggle_group.
                    attr('data-persistent-state', $(context.input).val());
            }).
            fail(function(jqxhr, status, error) {
                var context = this;
                $(document).trigger('ajax:complete');
                // Add a visual indication that something failed
                $('label.btn', context.toggle_group).
                    addClass('btn-danger').
                    removeClass('btn-info');
                // Select the button reflecting the last-saved-state
                $('label.btn input[value=' +
                  context.toggle_group.attr('data-persistent-state') +
                  ']', context.toggle_group).
                    button('toggle');
                if (jqxhr.responseJSON && jqxhr.responseJSON.errors) {
                    window.alert("Request failed: " +
                                 jqxhr.responseJSON.errors.join("; "));
                } else {
                    window.alert("Request failed.");
                }
            });
        $(document).trigger('ajax:send');
    });
});
