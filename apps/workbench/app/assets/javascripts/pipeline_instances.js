function run_pipeline_button_state() {
    var a = $('a.editable.required.editable-empty');
    if (a.length > 0) {
        $(".run-pipeline-button").addClass("disabled");
    }
    else {
        $(".run-pipeline-button").removeClass("disabled");
    }
}

$(document).on('editable:success', function(event, tag, response, newValue) {
    if ($('.run-pipeline-button').length == 0)
        return;
    tag = $(tag);
    if (tag.hasClass("required")) {
        if (newValue && newValue.trim() != "") {
            tag.removeClass("editable-empty");
            tag.parent().css("background-color", "");
            tag.parent().prev().css("background-color", "");
        }
        else {
            tag.addClass("editable-empty");
            tag.parent().css("background-color", "#ffdddd");
            tag.parent().prev().css("background-color", "#ffdddd");
        }
    }
    if (tag.attr('data-name')) {
        // Update other inputs representing the same piece of data
        $('[data-name="' + tag.attr('data-name') + '"]').html(newValue);
    }
    run_pipeline_button_state();
});

$(document).on('ready ajax:complete', function() {
    $('a.editable.required').each(function() {
        if (this.hasClass("editable-empty")) {
            this.parent().css("background-color", "#ffdddd");
            this.parent().prev().css("background-color", "#ffdddd");
        }
        else {
            this.parent().css("background-color", "");
            this.parent().prev().css("background-color", "");
        }
    });
    run_pipeline_button_state();
});
