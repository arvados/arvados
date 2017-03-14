package setup

const tmplNginxAPI = `server {
  listen 127.0.0.1:8000;
  server_name localhost-api;

  root /var/www/arvados-api/current/public;
  index  index.html index.htm index.php;

  passenger_enabled on;
  # if using RVM:
  passenger_ruby /usr/local/rvm/wrappers/default/ruby;

  # This value effectively limits the size of API objects users can
  # create, especially collections.  If you change this, you should
  # also ensure the following settings match it:
  # * "client_max_body_size" in the server section below
  # * "client_max_body_size" in the Workbench Nginx configuration (twice)
  # * "max_request_size" in the API server's application.yml file
  client_max_body_size 128m;
}

upstream api {
  server     127.0.0.1:8000  fail_timeout=10s;
}

upstream websockets {
  server     127.0.0.1:8100  fail_timeout=10s;
}

proxy_http_version 1.1;

# When Keep clients request a list of Keep services from the API server, the
# server will automatically return the list of available proxies if
# the request headers include X-External-Client: 1.  Following the example
# here, at the end of this section, add a line for each netmask that has
# direct access to Keep storage daemons to set this header value to 0.
geo $external_client {
  default        1;
  10.0.0.0/8     0;
  172.16.0.0/12  0;
  192.168.0.0/16 0;
}

server {
  listen       0.0.0.0:443 ssl;
  server_name  uuid_prefix.your.domain;

  ssl on;
  ssl_certificate     {{ keyOrDefault "arvados/api/sslCertPath" "/var/lib/arvados/certs/selfsigned.crt" }};
  ssl_certificate_key {{ keyOrDefault "arvados/api/sslKeyPath" "/var/lib/arvados/certs/selfsigned.key" }};

  index  index.html index.htm index.php;

  # Refer to the comment about this setting in the server section above.
  client_max_body_size 128m;

  location / {
    proxy_pass            http://api;
    proxy_redirect        off;
    proxy_connect_timeout 90s;
    proxy_read_timeout    300s;

    proxy_set_header      X-Forwarded-Proto https;
    proxy_set_header      Host $http_host;
    proxy_set_header      X-External-Client $external_client;
    proxy_set_header      X-Real-IP $remote_addr;
    proxy_set_header      X-Forwarded-For $proxy_add_x_forwarded_for;
  }
}

server {
  listen       0.0.0.0:443 ssl;
  server_name  ws.uuid_prefix.your.domain;

  ssl on;
  ssl_certificate     {{ keyOrDefault "arvados/ws/sslCertPath" "/var/lib/arvados/certs/selfsigned.crt" }};
  ssl_certificate_key {{ keyOrDefault "arvados/ws/sslKeyPath" "/var/lib/arvados/certs/selfsigned.key" }};

  index  index.html index.htm index.php;

  location / {
    proxy_pass            http://websockets;
    proxy_redirect        off;
    proxy_connect_timeout 90s;
    proxy_read_timeout    300s;

    proxy_set_header      Upgrade $http_upgrade;
    proxy_set_header      Connection "upgrade";
    proxy_set_header      Host $host;
    proxy_set_header      X-Real-IP $remote_addr;
    proxy_set_header      X-Forwarded-For $proxy_add_x_forwarded_for;
  }
}
`
