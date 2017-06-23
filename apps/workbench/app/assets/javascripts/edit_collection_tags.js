jQuery(function($){
  $(document).
    on('click', '.collection-tag-save, .collection-tag-cancel', function(event) {
        $('.edit-collection-tags').removeClass('disabled');
        $('#edit-collection-tags').attr("title", "Edit tags");
        $('.collection-tag-add').addClass('hide');
        $('.collection-tag-remove').addClass('hide');
        $('.collection-tag-save').addClass('hide');
        $('.collection-tag-cancel').addClass('hide');
        $('.collection-tag-field').prop("contenteditable", false);
    }).
    on('click', '.edit-collection-tags', function(event) {
        $('.edit-collection-tags').addClass('disabled');
        $('#edit-collection-tags').attr("title", "");
        $('.collection-tag-add').removeClass('hide');
        $('.collection-tag-remove').removeClass('hide');
        $('.collection-tag-save').removeClass('hide');
        $('.collection-tag-cancel').removeClass('hide');
        $('.collection-tag-field').prop("contenteditable", true);
        $('div').remove('.collection-tags-status-label');
    }).
    on('click', '.collection-tag-save', function(event) {
      var tag_data = {};
      var has_tags = false;

      var $tags = $(".collection-tags-table");
      $tags.find('tr').each(function (i, el) {
        var $tds = $(this).find('td');
        var $key = $tds.eq(1).text();
        if ($key && $key.trim().length > 0) {
          has_tags = true;
          tag_data[$key.trim()] = $tds.eq(2).text().trim();
        }
      });

      var to_send;
      if (has_tags == false) {
        to_send = {tag_data: "empty"}
      } else {
        to_send = {tag_data: tag_data}
      }

      $.ajax($(location).attr('pathname')+'/save_tags', {
          type: 'POST',
          data: to_send
      }).success(function(data, status, jqxhr) {
        $('.collection-tags-status').append('<div class="collection-tags-status-label alert alert-success"><p class="contain-align-left">Saved successfully.</p></div>');
      }).fail(function(jqxhr, status, error) {
        $('.collection-tags-status').append('<div class="collection-tags-status-label alert alert-danger"><p class="contain-align-left">We are sorry. There was an error saving tags. Please try again.</p></div>');
      });
    }).
    on('click', '.collection-tag-cancel', function(event) {
      $.ajax($(location).attr('pathname')+'/tags', {
          type: 'GET'
      });
    }).
    on('click', '.collection-tag-remove', function(event) {
      $(this).parents('tr').detach();
    }).
    on('click', '.collection-tag-add', function(event) {
      var $collection_tags = $(this).closest('.collection-tags-container');
      var $clone = $collection_tags.find('tr.hide').clone(true).removeClass('hide');
      $collection_tags.find('table').append($clone);
    }).
    on('keypress', '.collection-tag-field', function(event){
      return event.which != 13;
    });
});
