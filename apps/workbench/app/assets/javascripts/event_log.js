/*
 * This js establishes a websockets connection with the API Server.
 */

/* Subscribe to websockets event log.  Do nothing if already connected. */
function subscribeToEventLog () {
    // if websockets are not supported by browser, do not subscribe for events
    websocketsSupported = ('WebSocket' in window);
    if (websocketsSupported == false) {
        return;
    }

    // check if websocket connection is already stored on the window
    event_log_disp = $(window).data("arv-websocket");
    if (event_log_disp == null) {
        // need to create new websocket and event log dispatcher
        websocket_url = $('meta[name=arv-websocket-url]').attr("content");
        if (websocket_url == null)
            return;

        event_log_disp = new WebSocket(websocket_url);

        event_log_disp.onopen = onEventLogDispatcherOpen;
        event_log_disp.onmessage = onEventLogDispatcherMessage;

        // store websocket in window to allow reuse when multiple divs subscribe for events
        $(window).data("arv-websocket", event_log_disp);
    }
}

/* Send subscribe message to the websockets server.  Without any filters
   arguments, this subscribes to all events */
function onEventLogDispatcherOpen(event) {
    this.send('{"method":"subscribe"}');
}

/* Trigger event for all applicable elements waiting for this event */
function onEventLogDispatcherMessage(event) {
    parsedData = JSON.parse(event.data);
    object_uuid = parsedData.object_uuid;

    if (!object_uuid) {
        return;
    }

    // if there are any listeners for this object uuid or "all", trigger the event
    matches = ".arv-log-event-listener[data-object-uuid=\"" + object_uuid + "\"],.arv-log-event-listener[data-object-uuids~=\"" + object_uuid + "\"],.arv-log-event-listener[data-object-uuid=\"all\"],.arv-log-event-listener[data-object-kind=\"" + parsedData.object_kind + "\"]";
    $(matches).trigger('arv-log-event', parsedData);
}

/* Automatically connect if there are any elements on the page that want to
   receive event log events. */
$(document).on('ajax:complete ready', function() {
    var a = $('.arv-log-event-listener');
    if (a.length > 0) {
        subscribeToEventLog();
    }
});


function processLogLineForChart( logLine, jobGraphSeries, jobGraphData ) {
    var match = logLine.match(/(.*)crunchstat:(.*)-- interval(.*)/);
    if( match ) {
        var series = match[2].trim().split(' ')[0];
        if( $.inArray( series, jobGraphSeries) < 0 ) {
            jobGraphSeries.push(series);
        }
        var intervalData = match[3].trim().split(' ');
        var dt = parseFloat(intervalData[0]);
        var dsum = 0.0;
        for(var i=2; i < intervalData.length; i += 2 ) {
            dsum += parseFloat(intervalData[i]);
        }
        var datum = (dsum/dt).toFixed(4);
        var preamble = match[1].trim().split(' ');
        var timestamp = preamble[0].replace('_','T');
        var xpoints = $.grep( jobGraphData, function(e){ return e['t'] === timestamp; });
        if(xpoints.length) {
            // xpoints[0] is the x point that that matched the timestamp and so already existed: add the new datum
            xpoints[0][series] = datum;
        } else {
            var entry = { 't': timestamp };
            entry[series] = datum;
            jobGraphData.push( entry );
        }
    }
}

$(document).on('arv-log-event', '#log_graph_div', function(event, eventData) {
    if( eventData.properties.text ) {
        var series_length = jobGraphSeries.length;
        processLogLineForChart( eventData.properties.text, jobGraphSeries, jobGraphData);
        if( series_length < jobGraphSeries.length) {
            // series have changed, draw entirely new graph
            $('#log_graph_div').html('');
            jobGraph = Morris.Line({
                element: 'log_graph_div',
                data: jobGraphData,
                xkey: 't',
                ykeys: jobGraphSeries,
                labels: jobGraphSeries
            });
        } else {
            jobGraph.setData( jobGraphData );
        }
    }

} );