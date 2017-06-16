// On loading of a collection, enable the "lock" button and
// disable all file modification controls (upload, rename, delete)
$(document).
    ready(function(event) {
        $(".btn-collection-file-control").addClass("disabled");
        $(".btn-collection-rename-file-span").attr("title", "Unlock collection to rename file");
        $(".btn-collection-remove-file-span").attr("title", "Unlock collection to remove file");
        $(".btn-remove-selected-files").attr("title", "Unlock collection to remove selected files");
        $(".tab-pane-Upload").addClass("disabled");
        $(".tab-pane-Upload").attr("title", "Unlock collection to upload files");
        $("#Upload-tab").attr("data-toggle", "disabled");
    }).
    on('click', '.lock-collection-btn', function(event) {
        classes = $(event.target).attr('class')

        if (classes.indexOf("fa-lock") != -1) {
            // About to unlock; warn and get confirmation from user
            if (confirm("Adding, renaming, and deleting files changes the portable data hash. Are you sure you want to unlock the collection?")) {
                $(".lock-collection-btn").removeClass("fa-lock");
                $(".lock-collection-btn").addClass("fa-unlock");
                $(".lock-collection-btn").attr("title", "Lock collection to prevent editing files");
                $(".btn-collection-rename-file-span").attr("title", "");
                $(".btn-collection-remove-file-span").attr("title", "");
                $(".btn-collection-file-control").removeClass("disabled");
                $(".btn-remove-selected-files").attr("title", "");
                $(".tab-pane-Upload").removeClass("disabled");
                $(".tab-pane-Upload").attr("data-original-title", "");
                $("#Upload-tab").attr("data-toggle", "tab");

                $('.edit-collection-tags').removeClass('disabled');
            } else {
                // User clicked "no" and so do not unlock
            }
        } else {
            // Lock it back
            $(".lock-collection-btn").removeClass("fa-unlock");
            $(".lock-collection-btn").addClass("fa-lock");
            $(".lock-collection-btn").attr("title", "Unlock collection to edit files");
            $(".btn-collection-rename-file-span").attr("title", "Unlock collection to rename file");
            $(".btn-collection-remove-file-span").attr("title", "Unlock collection to remove file");
            $(".btn-collection-file-control").addClass("disabled");
            $(".btn-remove-selected-files").attr("title", "Unlock collection to remove selected files");
            $(".tab-pane-Upload").addClass("disabled");
            $(".tab-pane-Upload").attr("data-original-title", "Unlock collection to upload files");
            $("#Upload-tab").attr("data-toggle", "disabled");

            $('.edit-collection-tags').removeClass('hide');
            $('.edit-collection-tags').addClass('disabled');
            $('.collection-tag-add').addClass('hide');
            $('.collection-tag-remove').addClass('hide');
            $('.collection-tag-save').addClass('hide');
            $('.collection-tag-cancel').addClass('hide');
            $('.collection-tag-field').prop("contenteditable", false);
        }
    }).
    on('click', '.collection-tag-save, .collection-tag-cancel', function(event) {
        $('.edit-collection-tags').removeClass('hide');
        $('.collection-tag-add').addClass('hide');
        $('.collection-tag-remove').addClass('hide');
        $('.collection-tag-save').addClass('hide');
        $('.collection-tag-cancel').addClass('hide');
        $('.collection-tag-field').prop("contenteditable", false);
    }).
    on('click', '.edit-collection-tags', function(event) {
        $('.edit-collection-tags').addClass('hide');
        $('.collection-tag-add').removeClass('hide');
        $('.collection-tag-remove').removeClass('hide');
        $('.collection-tag-save').removeClass('hide');
        $('.collection-tag-cancel').removeClass('hide');
        $('.collection-tag-field').prop("contenteditable", true);
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
