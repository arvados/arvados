$(document).on('arv-log-event', '.arv-log-event-handler-append-logs', function(event, eventData){
    if (this != event.target) {
        // Not interested in events sent to child nodes.
        return;
    }
    var wasatbottom = ($(this).scrollTop() + $(this).height() >= this.scrollHeight);

    if (eventData.properties != null && eventData.properties.text != null) {
        if( eventData.prepend ) {
            $(this).prepend(eventData.properties.text);
        } else {
            $(this).append(eventData.properties.text);
        }
    }

    if (wasatbottom) {
        this.scrollTop = this.scrollHeight;
    }
});
