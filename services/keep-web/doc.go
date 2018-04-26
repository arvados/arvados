// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Keep-web provides read/write HTTP (WebDAV) access to files stored
// in Keep. It serves public data to anonymous and unauthenticated
// clients, and serves private data to clients that supply Arvados API
// tokens. It can be installed anywhere with access to Keep services,
// typically behind a web proxy that supports TLS.
//
// See http://doc.arvados.org/install/install-keep-web.html.
//
// Configuration
//
// The default configuration file location is
// /etc/arvados/keep-web/keep-web.yml.
//
// Example configuration file
//
//     Client:
//       APIHost: "zzzzz.arvadosapi.com:443"
//       AuthToken: ""
//       Insecure: false
//     Listen: :1234
//     AnonymousTokens:
//       - xxxxxxxxxxxxxxxxxxxx
//     AttachmentOnlyHost: ""
//     TrustAllContent: false
//
// Starting the server
//
// Start a server using the default config file
// /etc/arvados/keep-web/keep-web.yml:
//
//   keep-web
//
// Start a server using the config file /path/to/keep-web.yml:
//
//   keep-web -config /path/to/keep-web.yml
//
// Proxy configuration
//
// Keep-web does not support TLS natively. Typically, it is installed
// behind a proxy like nginx.
//
// Here is an example nginx configuration.
//
//	http {
//	  upstream keep-web {
//	    server localhost:1234;
//	  }
//	  server {
//	    listen *:443 ssl;
//	    server_name collections.example.com *.collections.example.com ~.*--collections.example.com;
//	    ssl_certificate /root/wildcard.example.com.crt;
//	    ssl_certificate_key /root/wildcard.example.com.key;
//	    location  / {
//	      proxy_pass http://keep-web;
//	      proxy_set_header Host $host;
//	      proxy_set_header X-Forwarded-For $remote_addr;
//	    }
//	  }
//	}
//
// It is not necessary to run keep-web on the same host as the nginx
// proxy. However, TLS is not used between nginx and keep-web, so
// intervening networks must be secured by other means.
//
// Anonymous downloads
//
// The "AnonymousTokens" configuration entry is an array of tokens to
// use when processing anonymous requests, i.e., whenever a web client
// does not supply its own Arvados API token via path, query string,
// cookie, or request header.
//
//   "AnonymousTokens":["xxxxxxxxxxxxxxxxxxxxxxx"]
//
// See http://doc.arvados.org/install/install-keep-web.html for examples.
//
// Download URLs
//
// The following "same origin" URL patterns are supported for public
// collections and collections shared anonymously via secret links
// (i.e., collections which can be served by keep-web without making
// use of any implicit credentials like cookies). See "Same-origin
// URLs" below.
//
//   http://collections.example.com/c=uuid_or_pdh/path/file.txt
//   http://collections.example.com/c=uuid_or_pdh/t=TOKEN/path/file.txt
//
// The following "multiple origin" URL patterns are supported for all
// collections:
//
//   http://uuid_or_pdh--collections.example.com/path/file.txt
//   http://uuid_or_pdh--collections.example.com/t=TOKEN/path/file.txt
//
// In the "multiple origin" form, the string "--" can be replaced with
// "." with identical results (assuming the downstream proxy is
// configured accordingly). These two are equivalent:
//
//   http://uuid_or_pdh--collections.example.com/path/file.txt
//   http://uuid_or_pdh.collections.example.com/path/file.txt
//
// The first form (with "--" instead of ".") avoids the cost and
// effort of deploying a wildcard TLS certificate for
// *.collections.example.com at sites that already have a wildcard
// certificate for *.example.com. The second form is likely to be
// easier to configure, and more efficient to run, on a downstream
// proxy.
//
// In all of the above forms, the "collections.example.com" part can
// be anything at all: keep-web itself ignores everything after the
// first "." or "--". (Of course, in order for clients to connect at
// all, DNS and any relevant proxies must be configured accordingly.)
//
// In all of the above forms, the "uuid_or_pdh" part can be either a
// collection UUID or a portable data hash with the "+" character
// optionally replaced by "-". (When "uuid_or_pdh" appears in the
// domain name, replacing "+" with "-" is mandatory, because "+" is
// not a valid character in a domain name.)
//
// In all of the above forms, a top level directory called "_" is
// skipped. In cases where the "path/file.txt" part might start with
// "t=" or "c=" or "_/", links should be constructed with a leading
// "_/" to ensure the top level directory is not interpreted as a
// token or collection ID.
//
// Assuming there is a collection with UUID
// zzzzz-4zz18-znfnqtbbv4spc3w and portable data hash
// 1f4b0bc7583c2a7f9102c395f4ffc5e3+45, the following URLs are
// interchangeable:
//
//   http://zzzzz-4zz18-znfnqtbbv4spc3w.collections.example.com/foo/bar.txt
//   http://zzzzz-4zz18-znfnqtbbv4spc3w.collections.example.com/_/foo/bar.txt
//   http://zzzzz-4zz18-znfnqtbbv4spc3w--collections.example.com/_/foo/bar.txt
//
// The following URLs are read-only, but otherwise interchangeable
// with the above:
//
//   http://1f4b0bc7583c2a7f9102c395f4ffc5e3-45--foo.example.com/foo/bar.txt
//   http://1f4b0bc7583c2a7f9102c395f4ffc5e3-45--.invalid/foo/bar.txt
//   http://collections.example.com/by_id/1f4b0bc7583c2a7f9102c395f4ffc5e3%2B45/foo/bar.txt
//   http://collections.example.com/by_id/zzzzz-4zz18-znfnqtbbv4spc3w/foo/bar.txt
//
// If the collection is named "MyCollection" and located in a project
// called "MyProject" which is in the home project of a user with
// username is "bob", the following read-only URL is also available
// when authenticating as bob:
//
//   http://collections.example.com/users/bob/MyProject/MyCollection/foo/bar.txt
//
// An additional form is supported specifically to make it more
// convenient to maintain support for existing Workbench download
// links:
//
//   http://collections.example.com/collections/download/uuid_or_pdh/TOKEN/foo/bar.txt
//
// A regular Workbench "download" link is also accepted, but
// credentials passed via cookie, header, etc. are ignored. Only
// public data can be served this way:
//
//   http://collections.example.com/collections/uuid_or_pdh/foo/bar.txt
//
// Collections can also be accessed (read-only) via "/by_id/X" where X
// is a UUID or portable data hash.
//
// Authorization mechanisms
//
// A token can be provided in an Authorization header:
//
//   Authorization: OAuth2 o07j4px7RlJK4CuMYp7C0LDT4CzR1J1qBE5Avo7eCcUjOTikxK
//
// A base64-encoded token can be provided in a cookie named "api_token":
//
//   Cookie: api_token=bzA3ajRweDdSbEpLNEN1TVlwN0MwTERUNEN6UjFKMXFCRTVBdm83ZUNjVWpPVGlreEs=
//
// A token can be provided in an URL-encoded query string:
//
//   GET /foo/bar.txt?api_token=o07j4px7RlJK4CuMYp7C0LDT4CzR1J1qBE5Avo7eCcUjOTikxK
//
// A suitably encoded token can be provided in a POST body if the
// request has a content type of application/x-www-form-urlencoded or
// multipart/form-data:
//
//   POST /foo/bar.txt
//   Content-Type: application/x-www-form-urlencoded
//   [...]
//   api_token=o07j4px7RlJK4CuMYp7C0LDT4CzR1J1qBE5Avo7eCcUjOTikxK
//
// If a token is provided in a query string or in a POST request, the
// response is an HTTP 303 redirect to an equivalent GET request, with
// the token stripped from the query string and added to a cookie
// instead.
//
// Indexes
//
// Keep-web returns a generic HTML index listing when a directory is
// requested with the GET method. It does not serve a default file
// like "index.html". Directory listings are also returned for WebDAV
// PROPFIND requests.
//
// Compatibility
//
// Client-provided authorization tokens are ignored if the client does
// not provide a Host header.
//
// In order to use the query string or a POST form authorization
// mechanisms, the client must follow 303 redirects; the client must
// accept cookies with a 303 response and send those cookies when
// performing the redirect; and either the client or an intervening
// proxy must resolve a relative URL ("//host/path") if given in a
// response Location header.
//
// Intranet mode
//
// Normally, Keep-web accepts requests for multiple collections using
// the same host name, provided the client's credentials are not being
// used. This provides insufficient XSS protection in an installation
// where the "anonymously accessible" data is not truly public, but
// merely protected by network topology.
//
// In such cases -- for example, a site which is not reachable from
// the internet, where some data is world-readable from Arvados's
// perspective but is intended to be available only to users within
// the local network -- the downstream proxy should configured to
// return 401 for all paths beginning with "/c=".
//
// Same-origin URLs
//
// Without the same-origin protection outlined above, a web page
// stored in collection X could execute JavaScript code that uses the
// current viewer's credentials to download additional data from
// collection Y -- data which is accessible to the current viewer, but
// not to the author of collection X -- from the same origin
// (``https://collections.example.com/'') and upload it to some other
// site chosen by the author of collection X.
//
// Attachment-Only host
//
// It is possible to serve untrusted content and accept user
// credentials at the same origin as long as the content is only
// downloaded, never executed by browsers. A single origin (hostname
// and port) can be designated as an "attachment-only" origin: cookies
// will be accepted and all responses will have a
// "Content-Disposition: attachment" header. This behavior is invoked
// only when the designated origin matches exactly the Host header
// provided by the client or downstream proxy.
//
//   "AttachmentOnlyHost":"domain.example:9999"
//
// Trust All Content mode
//
// In TrustAllContent mode, Keep-web will accept credentials (API
// tokens) and serve any collection X at
// "https://collections.example.com/c=X/path/file.ext".  This is
// UNSAFE except in the special case where everyone who is able write
// ANY data to Keep, and every JavaScript and HTML file written to
// Keep, is also trusted to read ALL of the data in Keep.
//
// In such cases you can enable trust-all-content mode.
//
//   "TrustAllContent":true
//
// When TrustAllContent is enabled, the only effect of the
// AttachmentOnlyHost flag is to add a "Content-Disposition:
// attachment" header.
//
//   "AttachmentOnlyHost":"domain.example:9999",
//   "TrustAllContent":true
//
// Depending on your site configuration, you might also want to enable
// the "trust all content" setting in Workbench. Normally, Workbench
// avoids redirecting requests to keep-web if they depend on
// TrustAllContent being enabled.
//
package main
