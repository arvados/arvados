/*
 * This js establishes a websockets connection with the API Server.
 */

/* The subscribe method takes a window element id and object id.
   Any log events for that particular object id are sent to that window element. */
function subscribeToEventLog (elementId) {
  // if websockets are not supported by browser, do not subscribe for events
  websocketsSupported = ('WebSocket' in window);
  if (websocketsSupported == false) {
    return;  
  }

  // grab websocket connection from window, if one exists
  event_log_disp = $(window).data("arv-websocket");
  if (event_log_disp == null) {
    // create the event log dispatcher
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

/* send subscribe message to the websockets server */
function onEventLogDispatcherOpen(event) {
  this.send('{"method":"subscribe"}');
}

/* trigger event for all applicable elements waiting for this event */
function onEventLogDispatcherMessage(event) {
  parsedData = JSON.parse(event.data);
  object_uuid = parsedData.object_uuid;

  // if there are any listeners for this object uuid or "all", trigger the event 
  matches = ".arv-log-event-listener[data-object-uuid=\"" + object_uuid + "\"],.arv-log-event-listener[data-object-uuids~=\"" + object_uuid + "\"],.arv-log-event-listener[data-object-uuid=\"all\"]";
  $(matches).trigger('arv-log-event', event.data);
}
