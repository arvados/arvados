// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

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
        }
    });
