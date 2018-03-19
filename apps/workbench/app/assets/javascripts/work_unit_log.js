// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

$(document).on('arv-log-event', '.arv-log-event-handler-append-logs', function(event, eventData){
    var wasatbottom, txt;
    if (this != event.target) {
        // Not interested in events sent to child nodes.
        return;
    }

    if (!('properties' in eventData)) {
        return;
    }

    txt = '';
    if ('text' in eventData.properties &&
       eventData.properties.text.length > 0) {
        txt += eventData.properties.text;
        if (txt.slice(txt.length-1) != "\n") {
            txt += "\n";
        }
    }
    if (eventData.event_type == 'update' &&
        eventData.object_uuid.indexOf("-dz642-") == 5 &&
        'old_attributes' in eventData.properties &&
        'new_attributes' in eventData.properties) {
        // Container update
        if (eventData.properties.old_attributes.state != eventData.properties.new_attributes.state) {
            var stamp = eventData.event_at + " ";
            switch(eventData.properties.new_attributes.state) {
            case "Queued":
                txt += stamp + "Container "+eventData.object_uuid+" was returned to the queue\n";
                break;
            case "Locked":
                txt += stamp + "Container "+eventData.object_uuid+" was taken from the queue by a dispatch process\n";
                break;
            case "Running":
                txt += stamp + "Container "+eventData.object_uuid+" started\n";
                break;
            case "Complete":
                txt += stamp + "Container "+eventData.object_uuid+" finished\n";
                break;
            case "Cancelled":
                txt += stamp + "Container "+eventData.object_uuid+" was cancelled\n";
                break;
            default:
                // Unknown state -- unexpected, might as well log it.
                txt += stamp + "Container "+eventData.object_uuid+" changed state to " +
                    eventData.properties.new_attributes.state + "\n";
                break;
            }
        }
    }

    if (txt == '') {
        return;
    }

    wasatbottom = (this.scrollTop + this.clientHeight >= this.scrollHeight);
    if (eventData.prepend) {
        $(this).prepend(txt);
    } else {
        $(this).append(txt);
    }
    if (wasatbottom) {
        this.scrollTop = this.scrollHeight;
    }
});
