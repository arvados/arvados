$(document).
    on('ajax:success', 'form.new_authorized_key',
       function(e, data, status, xhr) {
           $(e.target).parents('div.daxalert').fadeOut('slow', function() {
               $('<div class="alert alert-success daxalert"><button type="button" class="close" data-dismiss="alert">&times;</button><p>Key added.</p></div>').hide().replaceAll(this).fadeIn('slow');
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
               error_div.html('<p>Sorry, request failed.');
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
