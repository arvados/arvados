// Arvados-ws is an Arvados event feed for Websocket clients.
//
// See https://doc.arvados.org/install/install-arvados-ws.html.
//
// Usage
//
//     arvados-ws [-config /etc/arvados/ws/ws.yml] [-dump-config]
//
// Minimal configuration
//
//     Client:
//       APIHost: localhost:443
//     Listen: ":1234"
//     Postgres:
//       dbname: arvados_production
//       host: localhost
//       password: xyzzy
//       user: arvados
//
// Options
//
// -config path
//
// Load configuration from the given file instead of the default
// /etc/arvados/ws/ws.yml
//
// -dump-config
//
// Print the loaded configuration to stdout and exit.
//
// Logs
//
// Logs are printed to stderr, formatted as JSON.
//
// A log is printed each time a client connects or disconnects.
//
// Runtime status
//
// GET /debug.json responds with debug stats.
//
// GET /status.json responds with health check results and
// activity/usage metrics.
package main
