/*
 * This file establishes a websockets connection with the API Server.
 *
 * The subscribe method takes a window element id and object id. Any log
 * events for that particular object id are added to that window element.
 */

var event_log_disp;

function subscribeToEventLog (elementId, listeningOn) {
  // if websockets are not supported by browser, do not attempt to subscribe for events
  websocketsSupported = ('WebSocket' in window);
  if (websocketsSupported == false) {
    return;  
  }

  // grab websocket connection from window, if one exists
  event_log_disp = $(window).data("arv-websocket");
  if (event_log_disp == null) {
    // create the event log dispatcher
    event_log_disp = new WebSocket(sessionStorage.getItem("arv-websocket-url"));

    event_log_disp.onopen = function(event) { onEventLogDispatcherOpen(event) };
    event_log_disp.onmessage = function(event) { onEventLogDispatcherMessage(event) };

    // store websocket in window to allow reuse when multiple divs subscribe for events
    $(window).data("arv-websocket", event_log_disp);
  }

  // Add the elementId to listener map
  event_log_listener_map = $(window).data("event_log_listener_map");
  if (event_log_listener_map == null)
    event_log_listener_map = {};
  event_log_listener_map[elementId] = listeningOn;
  $(window).data("event_log_listener_map", event_log_listener_map);
}

function onEventLogDispatcherOpen(event) {
  event_log_disp.send('{"method":"subscribe"}');
}

// Check each of the entries in the listener map. If any are waiting for
// an event of this event's object, append to their registered element
function onEventLogDispatcherMessage(event) {
  event_log_listener_map = $(window).data("event_log_listener_map");

  parsedData = JSON.parse(event.data);
  event_uuid = parsedData.object_uuid;
  for (var key in event_log_listener_map) {
    value = event_log_listener_map[key];
    if (event_uuid === value) {
      matches = ".arv-log-event-listener[data-object-uuid=\"" + value + "\"]";
      $(matches).trigger('arv-log-event', event.data);
    }
  }
  // also trigger event for any listening for "all"
  $('.arv-log-event-listener[data-object-uuid="all"]').trigger('arv-log-event', event.data);
}
