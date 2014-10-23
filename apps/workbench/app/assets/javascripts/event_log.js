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

    if (object_uuid) {
        // if there are any listeners for this object uuid or "all", trigger the event
        matches = ".arv-log-event-listener[data-object-uuid=\"" + object_uuid + "\"],.arv-log-event-listener[data-object-uuids~=\"" + object_uuid + "\"],.arv-log-event-listener[data-object-uuid=\"all\"],.arv-log-event-listener[data-object-kind=\"" + parsedData.object_kind + "\"]";
        $(matches).trigger('arv-log-event', parsedData);
    }
}

/* Automatically connect if there are any elements on the page that want to
   received event log events. */
$(document).on('ajax:complete ready', function() {
  var a = $('.arv-log-event-listener');
  if (a.length > 0) {
    subscribeToEventLog();
  }
});
