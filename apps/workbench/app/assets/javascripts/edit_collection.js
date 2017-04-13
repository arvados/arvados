// On loading of a collection, enable the "lock" button and
// disable all file modification controls (upload, rename, delete)
$(document).
    ready(function(event) {
        $(".btn-collection-file-rename").addClass("disabled");
        $(".btn-collection-file-rename").attr("title", "Unlock collection to rename file");
        $(".btn-collection-file-remove").addClass("disabled");
        $(".btn-collection-file-remove").attr("title", "Unlock collection to remove file");
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
                $(".btn-collection-file-rename").removeClass("disabled");
                $(".btn-collection-file-rename").attr("data-original-title", "Edit name or path or both for this file");
                $(".btn-collection-file-remove").removeClass("disabled");
                $(".btn-collection-file-remove").attr("data-original-title", "Remove this file");
                $(".btn-remove-selected-files").attr("title", "");
                $(".tab-pane-Upload").removeClass("disabled");
                $(".tab-pane-Upload").attr("data-original-title", "");
                $("#Upload-tab").attr("data-toggle", "tab");
            } else {
                // User clicked "no" and so do not unlock
            }
        } else {
            // Lock it back
            $(".lock-collection-btn").removeClass("fa-unlock");
            $(".lock-collection-btn").addClass("fa-lock");
            $(".lock-collection-btn").attr("title", "Unlock collection to edit files");
            $(".btn-collection-file-rename").addClass("disabled");
            $(".btn-collection-file-rename").attr("data-original-title", "Unlock collection to rename file");
            $(".btn-collection-file-remove").addClass("disabled");
            $(".btn-collection-file-remove").attr("data-original-title", "Unlock collection to remove file");
            $(".btn-remove-selected-files").attr("title", "Unlock collection to remove selected files");
            $(".tab-pane-Upload").addClass("disabled");
            $(".tab-pane-Upload").attr("data-original-title", "Unlock collection to upload files");
            $("#Upload-tab").attr("data-toggle", "disabled");
        }
    });
