// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Arvados-ws exposes Arvados APIs (currently just one, the
// cache-invalidation event feed at "ws://.../websocket") to
// websocket clients.
//
// Installation
//
// See https://doc.arvados.org/install/install-ws.html.
//
// Developer info
//
// See https://dev.arvados.org/projects/arvados/wiki/Hacking_websocket_server.
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
// Enable additional logs by configuring:
//
//     LogLevel: debug
//
// Runtime status
//
// GET /debug.json responds with debug stats.
//
// GET /status.json responds with health check results and
// activity/usage metrics.
package main
