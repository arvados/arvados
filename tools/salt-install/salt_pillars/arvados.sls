# -*- coding: utf-8 -*-
# vim: ft=yaml
---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

# The variables commented out are the default values that the formula uses.
# The uncommented values are REQUIRED values. If you don't set them, running
# this formula will fail.
arvados:
  ### GENERAL CONFIG
  # version: '2.0.4'
  ## It makes little sense to disable this flag, but you can, if you want :)
  # use_upstream_repo: true

  ## Repo URL is built with grains values. If desired, it can be completely
  ## overwritten with the pillar parameter 'repo_url'
  # repo:
  #   humanname: Arvados Official Repository

  ## IMPORTANT!!!!!
  ## api, workbench and shell require some gems, so you need to make sure ruby
  ## and deps are installed in order to install and compile the gems.
  ## We default to `false` in these two variables as it's expected you already
  ## manage OS packages with some other tool and you don't want us messing up
  ## with your setup.
  ruby:
    ## We set these to `true` here for testing purposes.
    ## They both default to `false`.
    manage_ruby: true
    manage_gems_deps: true
    # pkg: ruby
    # gems_deps:
    #     - curl
    #     - g++
    #     - gcc
    #     - git
    #     - libcurl4
    #     - libcurl4-gnutls-dev
    #     - libpq-dev
    #     - libxml2
    #     - libxml2-dev
    #     - make
    #     - python3-dev
    #     - ruby-dev
    #     - zlib1g-dev

  # config:
  #   file: /etc/arvados/config.yml
  #   user: root
  ## IMPORTANT!!!!!
  ## If you're intalling any of the rails apps (api, workbench), the group
  ## should be set to that of the web server, usually `www-data`
  #   group: root
  #   mode: 640

  ### ARVADOS CLUSTER CONFIG
  cluster:
    name: fixme
    domain: arv.local

    database:
      # max concurrent connections per arvados server daemon
      # connection_pool_max: 32
      name: arvados
      host: 127.0.0.1
      password: changeme_arvados
      user: arvados
      encoding: en_US.utf8
      client_encoding: UTF8

    tls:
      # certificate: ''
      # key: ''
      # required to test with snakeoil certs
      insecure: true

    ### TOKENS
    tokens:
      system_root: changeme_system_root_token
      management: changeme_management_token
      rails_secret: changeme_rails_secret_token
      anonymous_user: changeme_anonymous_user_token

    ### KEYS
    secrets:
      blob_signing_key: changeme_blob_signing_key
      workbench_secret_key: changeme_workbench_secret_key
      dispatcher_access_key: changeme_dispatcher_access_key
      dispatcher_secret_key: changeme_dispatcher_secret_key
      keep_access_key: changeme_keep_access_key
      keep_secret_key: changeme_keep_secret_key

    AuditLogs:
      Section_to_ignore:
        - some_random_value

    ### VOLUMES
    ## This should usually match all your `keepstore` instances
    Volumes:
      # the volume name will be composed with
      # <cluster>-nyw5e-<volume>
      fixme-nyw5e-000000000000000:
        AccessViaHosts:
          http://keep0.fixme.arv.local:25107:
            ReadOnly: false
        Replication: 2
        Driver: Directory
        DriverParameters:
          Root: /tmp

    Users:
      NewUsersAreActive: true
      AutoAdminFirstUser: true
      AutoSetupNewUsers: true
      AutoSetupNewUsersWithRepository: true

    Services:
      Controller:
        ExternalURL: https://fixme.arv.local
        InternalURLs:
          http://localhost:8003: {}
      DispatchCloud:
        InternalURLs:
          http://fixme.arv.local:9006: {}
      Keepbalance:
        InternalURLs:
          http://fixme.arv.local:9005: {}
      Keepproxy:
        ExternalURL: https://keep.fixme.arv.local
        InternalURLs:
          http://localhost:25107: {}
      Keepstore:
        InternalURLs:
          http://keep0.fixme.arv.local:25107: {}
      RailsAPI:
        InternalURLs:
          http://localhost:8004: {}
      WebDAV:
        ExternalURL: https://collections.fixme.arv.local
        InternalURLs:
          http://localhost:9002: {}
      WebDAVDownload:
        ExternalURL: https://download.fixme.arv.local
      Websocket:
        ExternalURL: wss://ws.fixme.arv.local/websocket
        InternalURLs:
          http://localhost:8005: {}
      Workbench1:
        ExternalURL: https://workbench.fixme.arv.local
      Workbench2:
        ExternalURL: https://workbench2.fixme.arv.local

#  ### THESE ARE THE PACKAGES AND DAEMONS BASIC CONFIGS
#  #### API
#   api:
#     pkg:
#       name:
#         - arvados-api-server
#         - arvados-dispatch-cloud
#     gem:
#       name:
#         - arvados-cli
#     service:
#       name:
#         - nginx
#       port: 8004
#  #### CONTROLLER
#   controller:
#     pkg:
#       name: arvados-controller
#     service:
#       name: arvados-controller
#       port: 8003
#  #### DISPATCHER
#   dispatcher:
#     pkg:
#       name:
#         - crunch-dispatch-local
#       #   - arvados-dispatch-cloud
#       #   - crunch-dispatch-slurm
#     service:
#       name: crunch-dispatch-local
#       port: 9006
#  #### KEEPPROXY
#   keepproxy:
#     pkg:
#       name: keepproxy
#     service:
#       name: keepproxy
#       port: 25107
#  #### KEEPWEB
#   keepweb:
#     pkg:
#       name: keep-web
#     service:
#       name: keep-web
#     #   webdav
#       port: 9002
#  #### KEEPSTORE
#   keepstore:
#     pkg:
#       name: keepstore
#     service:
#       name: keepstore
#       port: 25107
#  #### GIT-HTTPD
#   githttpd:
#     pkg:
#       name: arvados-git-httpd
#     service:
#       name: arvados-git-httpd
#       port: 9001
#  #### SHELL
#   shell:
#     pkg:
#       name:
#         - arvados-client
#         - arvados-src
#         - libpam-arvados
#         - python-arvados-fuse
#         - python3-arvados-python-client
#         - python3-arvados-cwl-runner
#     gem:
#       name:
#         - arvados-cli
#         - arvados-login-sync
#  #### WORKBENCH
#   workbench:
#     pkg:
#       name: arvados-workbench
#     service:
#       name: nginx
#  #### WORKBENCH2
#   workbench2:
#     pkg:
#       name: arvados-workbench2
#     service:
#       name: nginx
#  ####  WEBSOCKET
#   websocket:
#     pkg:
#       name: arvados-ws
#     service:
#       name: arvados-ws
#       port: 8005
#  #### SSO
#   sso:
#     pkg:
#       name: arvados-sso
#     service:
#       name: arvados-sso
#       port: 8900

#  ## SALTSTACK FORMULAS TOFS configuration
#   https://template-formula.readthedocs.io/en/latest/TOFS_pattern.html
#   tofs:
#   #    The files_switch key serves as a selector for alternative
#   #    directories under the formula files directory. See TOFS pattern
#   #    doc for more info.
#   #    Note: Any value not evaluated by `config.get` will be used literally.
#   #    This can be used to set custom paths, as many levels deep as required.
#     files_switch:
#       - any/path/can/be/used/here
#       - id
#       - roles
#       - osfinger
#       - os
#       - os_family
#   #    All aspects of path/file resolution are customisable using the options below.
#   #    This is unnecessary in most cases; there are sensible defaults.
#   #    Default path: salt://< path_prefix >/< dirs.files >/< dirs.default >
#   #            I.e.: salt://arvados/files/default
#   #    path_prefix: template_alt
#   #    dirs:
#   #      files: files_alt
#   #      default: default_alt
#   #    The entries under `source_files` are prepended to the default source files
#   #    given for the state
#   #    source_files:
#   #      arvados-config-file-file-managed:
#   #        - 'example_alt.tmpl'
#   #        - 'example_alt.tmpl.jinja'
