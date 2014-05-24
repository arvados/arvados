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
    var $tag = $(tag);
    if ($('.run-pipeline-button').length == 0)
        return;
    if ($tag.hasClass("required")) {
        if (newValue && newValue.trim() != "") {
            $tag.removeClass("editable-empty");
            $tag.parent().css("background-color", "");
            $tag.parent().prev().css("background-color", "");
        }
        else {
            $tag.addClass("editable-empty");
            $tag.parent().css("background-color", "#ffdddd");
            $tag.parent().prev().css("background-color", "#ffdddd");
        }
    }
    if ($tag.attr('data-name')) {
        // Update other inputs representing the same piece of data
        $('.editable[data-name="' + $tag.attr('data-name') + '"]').
            editable('setValue', newValue);
    }
    run_pipeline_button_state();
});

$(document).on('ready ajax:complete', function() {
    $('a.editable.required').each(function() {
        var $tag = $(this);
        if ($tag.hasClass("editable-empty")) {
            $tag.parent().css("background-color", "#ffdddd");
            $tag.parent().prev().css("background-color", "#ffdddd");
        else {
            $tag.parent().css("background-color", "");
            $tag.parent().prev().css("background-color", "");
        }
        run_pipeline_button_state();
    } );

    $(document).on('ajax:complete ready', function() {
      var a = $('.arv-log-event-listener');
      if (a.length > 0) {
        $('.arv-log-event-listener').each(function() {
          subscribeToEventLog(this.id);
        });
      }
    });

    $(document).on('arv-log-event', '.arv-log-event-handler-append-logs', function(event, eventData){
      parsedData = JSON.parse(eventData);

      propertyText = undefined

      properties = parsedData.properties;
      if (properties !== null) {
        propertyText = properties.text;
      }

      if (propertyText !== undefined) {
        $(this).append(propertyText + "<br/>");
      } else {
        $(this).append(parsedData.summary + "<br/>");
      }
    });

})();
