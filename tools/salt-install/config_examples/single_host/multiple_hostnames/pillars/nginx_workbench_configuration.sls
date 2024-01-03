---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- import_yaml "ssl_key_encrypted.sls" as ssl_key_encrypted_pillar %}

### ARVADOS
arvados:
  config:
    group: www-data

### NGINX
nginx:
  ### SITES
  servers:
    managed:
      ### DEFAULT
      arvados_workbench_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench.__DOMAIN__
            - listen:
              - 80
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_workbench_ssl.conf:
        enabled: true
        overwrite: true
        requires:
          __CERT_REQUIRES__
        config:
          # Maps WB1 '/actions?uuid=X' URLs to their equivalent on WB2
          - 'map $request_uri $actions_redirect':
            - '~^/actions\?uuid=(.*-4zz18-.*)': '/collections/$1'
            - '~^/actions\?uuid=(.*-j7d0g-.*)': '/projects/$1'
            - '~^/actions\?uuid=(.*-tpzed-.*)': '/projects/$1'
            - '~^/actions\?uuid=(.*-7fd4e-.*)': '/workflows/$1'
            - '~^/actions\?uuid=(.*-xvhdp-.*)': '/processes/$1'
            - '~^/actions\?uuid=(.*)': '/'
            - default: 0

          - server:
            - server_name: workbench.__DOMAIN__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm

    # REDIRECTS FROM WORKBENCH 1 TO WORKBENCH 2

    # Paths that are not redirected because wb1 and wb2 have similar enough paths
    # that a redirect is pointless and would create a redirect loop.
    # rewrite ^/api_client_authorizations.* /api_client_authorizations redirect;
    # rewrite ^/repositories.* /repositories redirect;
    # rewrite ^/links.* /links redirect;
    # rewrite ^/projects.* /projects redirect;
    # rewrite ^/trash /trash redirect;

            # WB1 '/actions?uuid=X' URL Redirects
            - 'if ($actions_redirect)':
              - return: '301 $actions_redirect'

    # Redirects that include a uuid
            - rewrite: '^/work_units/(.*) /processes/$1 redirect'
            - rewrite: '^/container_requests/(.*) /processes/$1 redirect'
            - rewrite: '^/users/(.*) /user/$1 redirect'
            - rewrite: '^/groups/(.*) /group/$1 redirect'

    # Special file download redirects
            - 'if ($arg_disposition = attachment)':
              - rewrite: '^/collections/([^/]*)/(.*) /?redirectToDownload=/c=$1/$2? redirect'

            - 'if ($arg_disposition = inline)':
              - rewrite: '^/collections/([^/]*)/(.*) /?redirectToPreview=/c=$1/$2? redirect'

    # Redirects that go to a roughly equivalent page
            - rewrite: '^/virtual_machines.* /virtual-machines-admin redirect'
            - rewrite: '^/users/.*/virtual_machines /virtual-machines-user redirect'
            - rewrite: '^/authorized_keys.* /ssh-keys-admin redirect'
            - rewrite: '^/users/.*/ssh_keys /ssh-keys-user redirect'
            - rewrite: '^/containers.* /all_processes redirect'
            - rewrite: '^/container_requests /all_processes redirect'
            - rewrite: '^/job.* /all_processes redirect'
            - rewrite: '^/users/link_account /link_account redirect'
            - rewrite: '^/keep_services.* /keep-services redirect'
            - rewrite: '^/trash_items.* /trash redirect'

    # Redirects that don't have a good mapping and
    # just go to root.
            - rewrite: '^/themes.* / redirect'
            - rewrite: '^/keep_disks.* / redirect'
            - rewrite: '^/user_agreements.* / redirect'
            - rewrite: '^/nodes.* / redirect'
            - rewrite: '^/humans.* / redirect'
            - rewrite: '^/traits.* / redirect'
            - rewrite: '^/sessions.* / redirect'
            - rewrite: '^/logout.* / redirect'
            - rewrite: '^/logged_out.* / redirect'
            - rewrite: '^/current_token / redirect'
            - rewrite: '^/logs.* / redirect'
            - rewrite: '^/factory_jobs.* / redirect'
            - rewrite: '^/uploaded_datasets.* / redirect'
            - rewrite: '^/specimens.* / redirect'
            - rewrite: '^/pipeline_templates.* / redirect'
            - rewrite: '^/pipeline_instances.* / redirect'

            - location /:
              - root: /var/www/arvados-workbench2/workbench2
              - try_files: '$uri $uri/ /index.html'
              - 'if (-f $document_root/maintenance.html)':
                - return: 503
            - location /config.json:
              - return: {{ "200 '" ~ '{"API_HOST":"__DOMAIN__:__CONTROLLER_EXT_SSL_PORT__"}' ~ "'" }}
            - include: snippets/ssl_hardening_default.conf
            - ssl_certificate: __CERT_PEM__
            - ssl_certificate_key: __CERT_KEY__
            {%- if ssl_key_encrypted_pillar.ssl_key_encrypted.enabled %}
            - ssl_password_file: {{ '/run/arvados/' | path_join(ssl_key_encrypted_pillar.ssl_key_encrypted.privkey_password_filename) }}
            {%- endif %}
            - access_log: /var/log/nginx/workbench2.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/workbench2.__DOMAIN__.error.log
