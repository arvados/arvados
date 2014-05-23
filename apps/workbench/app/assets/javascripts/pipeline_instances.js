(function() {
    var run_pipeline_button_state = function() {
        var a = $('a.editable.required.editable-empty');
        if (a.length > 0) {
            $("#run-pipeline-button").addClass("disabled");
        }
        else {
            $("#run-pipeline-button").removeClass("disabled");
        }
    }

    $.fn.editable.defaults.success = function (response, newValue) {
        var tag = $(this);
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
        run_pipeline_button_state();
    }

    $(window).on('load', function() {
        var a = $('a.editable.required');
        for (var i = 0; i < a.length; i++) {
            var tag = $(a[i]);
            if (tag.hasClass("editable-empty")) {
                tag.parent().css("background-color", "#ffdddd");
                tag.parent().prev().css("background-color", "#ffdddd");
            }
            else {
                tag.parent().css("background-color", "");
                tag.parent().prev().css("background-color", "");
            }
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
