// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

jQuery(function($){
    $(document).on('click', '.toggle-persist button', function() {
        var toggle_group = $(this).parents('[data-remote-href]').first();
        var want_persist = !toggle_group.find('button').hasClass('active');
        var want_state = want_persist ? 'persistent' : 'cache';
        toggle_group.find('button').
            toggleClass('active', want_persist).
            html(want_persist ? 'Persistent' : 'Cache');
        $.ajax(toggle_group.attr('data-remote-href'),
               {dataType: 'json',
                type: 'POST',
                data: {
                    value: want_state
                },
                context: {
                    toggle_group: toggle_group,
                    want_state: want_state,
                    button: this
                }
               }).
            done(function(data, status, jqxhr) {
                var context = this;
                // Remove "danger" status in case a previous action failed
                $('.btn-danger', context.toggle_group).
                    addClass('btn-info').
                    removeClass('btn-danger');
                // Update last-saved-state
                context.toggle_group.
                    attr('data-persistent-state', context.want_state);
            }).
            fail(function(jqxhr, status, error) {
                var context = this;
                var saved_state;
                // Add a visual indication that something failed
                $(context.button).
                    addClass('btn-danger').
                    removeClass('btn-info');
                // Change to the last-saved-state
                saved_state = context.toggle_group.attr('data-persistent-state');
                $(context.button).
                    toggleClass('active', saved_state == 'persistent').
                    html(saved_state == 'persistent' ? 'Persistent' : 'Cache');

                if (jqxhr.readyState == 0 || jqxhr.status == 0) {
                    // Request cancelled due to page reload.
                    // Displaying an alert would be rather annoying.
                } else if (jqxhr.responseJSON && jqxhr.responseJSON.errors) {
                    window.alert("Request failed: " +
                                 jqxhr.responseJSON.errors.join("; "));
                } else {
                    window.alert("Request failed.");
                }
            });
    });
});
