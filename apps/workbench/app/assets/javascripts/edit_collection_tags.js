// On loading of a collection, enable the "lock" button and
// disable all file modification controls (upload, rename, delete)
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
    on('click', '.collection-tag-save', function(e){
      var tag_data = {};
      var $tags = $(".collection-tags-table");
      $tags.find('tr').each(function (i, el) {
        var $tds = $(this).find('td');
        var $key = $tds.eq(1).text();
        if ($key && $key.trim().length > 0) {
          tag_data[$key.trim()] = $tds.eq(2).text().trim();
        }
      });

      if(jQuery.isEmptyObject(tag_data)){
        tag_data["empty"]=true
      } else {
        tag_data = {tag_data}
      }

      $.ajax($(location).attr('pathname')+'/save_tags', {
          type: 'POST',
          data: tag_data
      }).success(function(data, status, jqxhr) {
        $('.collection-tags-status').append('<div class="collection-tags-status-label alert alert-success"><p class="contain-align-left">Saved successfully.</p></div>');
      }).fail(function(jqxhr, status, error) {
        $('.collection-tags-status').append('<div class="collection-tags-status-label alert alert-danger"><p class="contain-align-left">We are sorry. There was an error saving tags. Please try again.</p></div>');
      });
    }).
    on('click', '.collection-tag-cancel', function(e){
      $.ajax($(location).attr('pathname')+'/tags', {
          type: 'GET'
      });
    });

jQuery(function($){
  $(document).on('click', '.collection-tag-remove', function(e) {
    $(this).parents('tr').detach();
  });

  $(document).on('click', '.collection-tag-add', function(e) {
    var $collection_tags = $(this).closest('.collection-tags-container');
    var $clone = $collection_tags.find('tr.hide').clone(true).removeClass('hide');
    $collection_tags.find('table').append($clone);
  });
});
