/*
 * This file establishes a websockets connection with the API Server.
 *
 * The subscribe method takes a window element id and object id. Any log
 * events for that particular object id are added to that window element.
 */

var event_log_disp;

function subscribeToEventLog (url, uuid, elementId) {
  // if websockets are not supported by browser, do not attempt to subscribe for events
  websocketsSupported = ('WebSocket' in window);
  if (websocketsSupported == false) {
    return;  
  }

  // create the event log dispatcher
  event_log_disp = new WebSocket(url);

  event_log_disp.onopen = function(event) { onEventLogDispatcherOpen(event) };
  event_log_disp.onmessage = function(event) { onEventLogDispatcherMessage(event) };

  // Add the elementId to listener map
  event_log_listener_map = JSON.parse(sessionStorage.getItem("event_log_listener_map"));
  if (event_log_listener_map == null)
    event_log_listener_map = {};
  delete event_log_listener_map[elementId];
  event_log_listener_map[elementId] = uuid;

  sessionStorage.removeItem("event_log_listener_map");
  sessionStorage.setItem("event_log_listener_map", JSON.stringify(event_log_listener_map));
}

function onEventLogDispatcherOpen(event) {
  event_log_disp.send('{"method":"subscribe"}');
}

function onEventLogDispatcherMessage(event) {
  event_log_listener_map = JSON.parse(sessionStorage.getItem("event_log_listener_map"));

  for (var key in event_log_listener_map) {
    value = event_log_listener_map[key];

    eventData = JSON.parse(event.data);
    if (value === eventData.object_uuid) {
      $('#'+key).append(eventData.summary + "&#13;&#10;");
    }
  }
}
