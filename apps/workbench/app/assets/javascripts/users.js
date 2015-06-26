$(document).
    on('notifications:recount',
       function() {
           var menu = $('.notification-menu');
           n = $('.notification', menu).not('.empty').length;
           $('.notification-count', menu).html(n>0 ? n : '');
       }).
    on('ajax:success', 'form.new_authorized_key',
       function(e, data, status, xhr) {
           $(e.target).parents('.notification').eq(0).fadeOut('slow', function() {
               $('<li class="alert alert-success daxalert">SSH key added.</li>').hide().replaceAll(this).fadeIn('slow');
               $(document).trigger('notifications:recount');
           });
       }).
    on('ajax:complete', 'form.new_authorized_key',
       function(e, data, status, xhr) {
           $($('input[name=disable_element]', e.target).val()).
               fadeTo(200, 1.0);
       }).
    on('ajax:error', 'form.new_authorized_key',
       function(e, xhr, status, error) {
           var error_div;
           response = $.parseJSON(xhr.responseText);
           error_div = $(e.target).parent().find('div.ajax-errors');
           if (error_div.length == 0) {
               $(e.target).parent().append('<div class="alert alert-error ajax-errors"></div>');
               error_div = $(e.target).parent().find('div.ajax-errors');
           }
           if (response.errors) {
               error_div.html($('<p/>').text(response.errors).html());
           } else {
               error_div.html('<p>Sorry, request failed.</p>');
           }
           error_div.show();
           $($('input[name=disable_element]', e.target).val()).
               fadeTo(200, 1.0);
       }).
    on('click', 'form[data-remote] input[type=submit]',
       function(e) {
           $(e.target).parents('form').eq(0).parent().find('div.ajax-errors').html('').hide();
           $($(e.target).
             parents('form').
             find('input[name=disable_element]').
             val()).
               fadeTo(200, 0.3);
           return true;
       });
