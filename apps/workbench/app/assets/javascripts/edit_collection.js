// On loading of a collection, enable the "lock" button and
// disable all file modification controls (upload, rename, delete)
$(document).
    ready(function(event) {
        $(".btn-collection-file-control").addClass("disabled");
        $(".tab-pane-Upload").addClass("disabled");
        $("#Upload-tab").attr("data-toggle", "disabled");
    }).
    on('click', '.lock-collection-btn', function(event) {
        classes = $(event.target).attr('class')

        if (classes.indexOf("fa-lock") != -1) {
            // About to unlock; warn the user
            if (confirm("Adding, renaming, and deleting files changes the portalble data hash. Are you sure you want to unlock the collection?")) {
                $(".lock-collection-btn").removeClass("fa-lock");
                $(".lock-collection-btn").addClass("fa-unlock");
                $(".btn-collection-file-control").removeClass("disabled");
                $(".tab-pane-Upload").removeClass("disabled");
                $("#Upload-tab").attr("data-toggle", "tab");
            } else {
                // User clicked "no" and so do not unlock
            }
        } else {
            // Lock it back
            $(".lock-collection-btn").removeClass("fa-unlock");
            $(".lock-collection-btn").addClass("fa-lock");
            $(".btn-collection-file-control").addClass("disabled");
            $(".tab-pane-Upload").addClass("disabled");
            $("#Upload-tab").attr("data-toggle", "disabled");
        }
    });
