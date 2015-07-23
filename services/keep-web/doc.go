// Keep-web provides read-only HTTP access to files stored in Keep. It
// serves public data to anonymous and unauthenticated clients, and
// accepts authentication via Arvados tokens. It can be installed
// anywhere with access to Keep services, typically behind a web proxy
// that provides SSL support.
//
// Given that this amounts to a web hosting service for arbitrary
// content, it is vital to ensure that at least one of the following is
// true:
//
// Usage
//
// Listening:
//
//   keep-web -address=:1234
//
// Start an HTTP server on port 1234.
//
//   keep-web -address=1.2.3.4:1234
//
// Start an HTTP server on port 1234, on the interface with IP address 1.2.3.4.
//
// Keep-web does not support SSL natively. Typically, it is installed
// behind a proxy like nginx.
//
package main

// TODO(TC): Implement
//
// Trusted content
//
// Normally, Keep-web is installed using a wildcard DNS entry and a
// wildcard HTTPS certificate, serving data from collection X at
// ``https://X.dl.example.com/path/file.ext''.
//
// It will also serve publicly accessible data at
// ``https://dl.example.com/collections/X/path/file.txt'', but it does not
// accept any kind of credentials at paths like these.
//
// In "trust all content" mode, Keep-web will accept credentials (API
// tokens) and serve any collection X at
// "https://dl.example.com/collections/X/path/file.ext".  This is
// UNSAFE except in the special case where everyone who is able write
// ANY data to Keep, and every JavaScript and HTML file written to
// Keep, is also trusted to read ALL of the data in Keep.
//
// In such cases you can enable trust-all-content mode.
//
//   keep-web -trust-all-content [...]
//
// In the general case, this should not be enabled: A web page stored
// in collection X can execute JavaScript code that uses the current
// viewer's credentials to download additional data -- data which is
// accessible to the current viewer, but not to the author of
// collection X -- from the same origin (``https://dl.example.com/'')
// and upload it to some other site chosen by the author of collection
// X.
