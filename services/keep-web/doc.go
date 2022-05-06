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
// The default cluster configuration file location is
// /etc/arvados/config.yml.
//
// Example configuration file
//
//   Clusters:
//     zzzzz:
//       SystemRootToken: ""
//       Services:
//         Controller:
//           ExternalURL: "https://example.com"
//           Insecure: false
//         WebDAV:
//           InternalURLs:
//             "http://:1234/": {}
//         WebDAVDownload:
//           InternalURLs:
//             "http://:1234/": {}
//           ExternalURL: "https://download.example.com/"
//       Users:
//         AnonymousUserToken: "xxxxxxxxxxxxxxxxxxxx"
//       Collections:
//         TrustAllContent: false
//
// Starting the server
//
// Start a server using the default config file
// /etc/arvados/config.yml:
//
//   keep-web
//
// Start a server using the config file /path/to/config.yml:
//
//   keep-web -config /path/to/config.yml
//
// Proxy configuration
//
// Typically, keep-web is installed behind a proxy like nginx.
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
// proxy. However, if TLS is not used between nginx and keep-web, the
// intervening networks must be secured by other means.
//
// Anonymous downloads
//
// The "Users.AnonymousUserToken" configuration entry used when
// when processing anonymous requests, i.e., whenever a web client
// does not supply its own Arvados API token via path, query string,
// cookie, or request header.
//
//   Clusters:
//     zzzzz:
//       Users:
//         AnonymousUserToken: "xxxxxxxxxxxxxxxxxxxxxxx"
//
// See http://doc.arvados.org/install/install-keep-web.html for examples.
//
// Download URLs
//
// See http://doc.arvados.org/api/keep-web-urls.html
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
//   Clusters:
//     zzzzz:
//       Services:
//         WebDAVDownload:
//           ExternalURL: "https://domain.example:9999"
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
//   Clusters:
//     zzzzz:
//       Collections:
//         TrustAllContent: true
//
// When TrustAllContent is enabled, the only effect of the
// Attachment-Only host setting is to add a "Content-Disposition:
// attachment" header.
//
//   Clusters:
//     zzzzz:
//       Services:
//         WebDAVDownload:
//           ExternalURL: "https://domain.example:9999"
//       Collections:
//         TrustAllContent: true
//
// Depending on your site configuration, you might also want to enable
// the "trust all content" setting in Workbench. Normally, Workbench
// avoids redirecting requests to keep-web if they depend on
// TrustAllContent being enabled.
//
// Metrics
//
// Keep-web exposes request metrics in Prometheus text-based format at
// /metrics. The same information is also available as JSON at
// /metrics.json.
//
package keepweb
