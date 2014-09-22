function run_pipeline_button_state() {
    var a = $('a.editable.required.editable-empty,input.form-control.required[value=]');
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
        }
        else {
            $tag.parent().css("background-color", "");
            $tag.parent().prev().css("background-color", "");
        }
    });
    run_pipeline_button_state();
});

$(document).on('arv-log-event', '.arv-log-event-handler-append-logs', function(event, eventData){
    var wasatbottom = ($(this).scrollTop() + $(this).height() >=
                       this.scrollHeight);
    var parsedData = JSON.parse(eventData);
    var propertyText = undefined;
    var properties = parsedData.properties;

    if (properties !== null) {
        propertyText = properties.text;
    }
    if (propertyText !== undefined) {
        propertyText = propertyText.
            replace(/\n$/, '').
            replace(/\n/g, '<br/>');
        $(this).append(propertyText + "<br/>");
    } else if (parsedData.summary !== undefined) {
        if (parsedData.summary.match(/^update of [-a-z0-9]{27}$/))
            ; // Not helpful.
        else
            $(this).append(parsedData.summary + "<br/>");
    }
    if (wasatbottom)
        this.scrollTop = this.scrollHeight;
}).on('arv:pane:loaded', '#Logs,#Log', function(){
    $('.arv-log-event-handler-append-logs', this).each(function() {
        this.scrollTop = this.scrollHeight;
        $(this).closest('.tab-pane').on('arv:pane:reload', function(e) {
            // Do not let this tab auto-refresh.
            e.stopPropagation();
        });
    });
}).on('ready ajax:complete', function(){
    $(".arv-log-event-listener[data-object-uuids-live]").each(function() {
        // Look at data-object-uuid attribute of elements matching
        // given selector, so the event listener can listen for events
        // that appeared on the page via ajax.
        var $listener = $(this);
        var have_uuids = '' + $listener.attr('data-object-uuids');
        $($listener.attr('data-object-uuids-live')).each(function() {
            var this_uuid = $(this).attr('data-object-uuid');
            if (have_uuids.indexOf(this_uuid) == -1) {
                have_uuids = have_uuids + ' ' + this_uuid;
            }
        });
        $listener.attr('data-object-uuids', have_uuids);
    });
});

var showhide_compare = function() {
    var form = $('form#compare')[0];
    $('input[type=hidden][name="uuids[]"]', form).remove();
    $('input[type=submit]', form).prop('disabled',true).show();
    var checked_inputs = $('[data-object-uuid*=-d1hrv-] input[name="uuids[]"]:checked');
    if (checked_inputs.length >= 2 && checked_inputs.length <= 3) {
        checked_inputs.each(function(){
            if(this.checked) {
                $('input[type=submit]', form).prop('disabled',false).show();
                $(form).append($('<input type="hidden" name="uuids[]"/>').val(this.value));
            }
        });
    }
};
$('[data-object-uuid*=-d1hrv-] input[name="uuids[]"]').on('click', showhide_compare);
showhide_compare();

setInterval(function(){
    if ($('[data-pipeline-state=RunningOnServer],[data-pipeline-state=RunningOnClient]').length > 0) {
        $('#Components-tab,#Graph-tab,#pipeline-instance-tab-buttons').trigger('arv:pane:reload');
    }
}, 15000);
