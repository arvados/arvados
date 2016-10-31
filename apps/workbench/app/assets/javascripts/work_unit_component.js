$(document).
    on('click', '.component-detail-panel', function(event) {
      var href = $($(event.target).attr('href'));
      if ($(href).attr("class").split(' ').indexOf("in") == -1) {
        return;   // collapsed; nothing more to do
      }

      var content_div = href.find('.work-unit-component-detail-body');
      content_div.html('<div class="spinner spinner-32px col-sm-1"></div>');
      var content_url = href.attr('content-url');
      var action_data = href.attr('action-data');
      $.ajax(content_url, {dataType: 'html', type: 'POST', data: {action_data: action_data}}).
          done(function(data, status, jqxhr) {
              content_div.html(data);
          }).fail(function(jqxhr, status, error) {
              content_div.html(error);
          });
      });
