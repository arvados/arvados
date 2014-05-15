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

  // grab websocket connection from window, if one exists
  event_log_disp = $(window).data("arv-websocket");
  if (event_log_disp == null) {
    // create the event log dispatcher
    event_log_disp = new WebSocket(url);

    event_log_disp.onopen = function(event) { onEventLogDispatcherOpen(event) };
    event_log_disp.onmessage = function(event) { onEventLogDispatcherMessage(event) };

    // store websocket in window to allow reuse when multiple divs subscribe for events
    $(window).data("arv-websocket", event_log_disp);

    // detach elementId when user leaves this window
    window.onbeforeunload = function (event) {detachElement(event, elementId);}
  }

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

// Check each of the entries in the listener map. If any are waiting for
// an event of this event's object, append to their registered element
function onEventLogDispatcherMessage(event) {
  event_log_listener_map = JSON.parse(sessionStorage.getItem("event_log_listener_map"));

  for (var key in event_log_listener_map) {
    value = event_log_listener_map[key];

    new_properties = "";
    eventData = JSON.parse(event.data);
    if (value === eventData.object_uuid) {
      properties = eventData.properties;
      if (properties !== null) {
        new_attributes = properties.new_attributes;
        if (new_attributes !== null) {
          new_properties = JSON.stringify(properties.new_attributes);
        }
      }

      // append to the registered element
      $('#'+key).append(eventData.summary + " " + new_properties + "&#13;&#10;");
    }
  }
}

/**
 * When leaving this window, remove elementId from map
 */
function detachElement(event, elementId) {
  event_log_listener_map = JSON.parse(sessionStorage.getItem("event_log_listener_map"));
  if (event_log_listener_map !== null) {
    delete event_log_listener_map[elementId];
  }

  // According to documenation, browser will close the connection when page is left.
  // However, there are suggestions of inconsistency; so to be safe, close explicitly.
  if (event_log_disp !== null) {
    event_log_disp.close();
  }
}
