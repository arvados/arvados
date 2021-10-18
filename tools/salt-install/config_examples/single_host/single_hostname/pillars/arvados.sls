---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# The variables commented out are the default values that the formula uses.
# The uncommented values are REQUIRED values. If you don't set them, running
# this formula will fail.
arvados:
  ### GENERAL CONFIG
  version: '__VERSION__'
  ## It makes little sense to disable this flag, but you can, if you want :)
  # use_upstream_repo: true

  ## Repo URL is built with grains values. If desired, it can be completely
  ## overwritten with the pillar parameter 'repo_url'
  # repo:
  #   humanname: Arvados Official Repository

  release: __RELEASE__

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
    name: __CLUSTER__
    domain: __DOMAIN__

    database:
      # max concurrent connections per arvados server daemon
      # connection_pool_max: 32
      name: __CLUSTER___arvados
      host: 127.0.0.1
      password: "__DATABASE_PASSWORD__"
      user: __CLUSTER___arvados
      encoding: en_US.utf8

    tls:
      # certificate: ''
      # key: ''
      # When using arvados-snakeoil certs set insecure: true
      insecure: true

    ### TOKENS
    tokens:
      system_root: __SYSTEM_ROOT_TOKEN__
      management: __MANAGEMENT_TOKEN__
      anonymous_user: __ANONYMOUS_USER_TOKEN__
      rails_secret: YDLxHf4GqqmLXYAMgndrAmFEdqgC0sBqX7TEjMN2rw9D6EVwgx

    ### KEYS
    secrets:
      blob_signing_key: __BLOB_SIGNING_KEY__
      workbench_secret_key: __WORKBENCH_SECRET_KEY__

    Login:
      Test:
        Enable: true
        Users:
          __INITIAL_USER__:
            Email: __INITIAL_USER_EMAIL__
            Password: __INITIAL_USER_PASSWORD__

    ### VOLUMES
    ## This should usually match all your `keepstore` instances
    Volumes:
      # the volume name will be composed with
      # <cluster>-nyw5e-<volume>
      __CLUSTER__-nyw5e-000000000000000:
        AccessViaHosts:
          'http://__HOSTNAME_INT__:25107':
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
        ExternalURL: 'https://__HOSTNAME_EXT__:__CONTROLLER_EXT_SSL_PORT__'
        InternalURLs:
          'http://__HOSTNAME_INT__:8003': {}
      Keepproxy:
        ExternalURL: 'https://__HOSTNAME_EXT__:__KEEP_EXT_SSL_PORT__'
        InternalURLs:
          'http://__HOSTNAME_INT__:25100': {}
      Keepstore:
        InternalURLs:
          'http://__HOSTNAME_INT__:25107': {}
      RailsAPI:
        InternalURLs:
          'http://__HOSTNAME_INT__:8004': {}
      WebDAV:
        ExternalURL: 'https://__HOSTNAME_EXT__:__KEEPWEB_EXT_SSL_PORT__'
        InternalURLs:
          'http://__HOSTNAME_INT__:9003': {}
      WebDAVDownload:
        ExternalURL: 'https://__HOSTNAME_EXT__:__KEEPWEB_EXT_SSL_PORT__'
      WebShell:
        ExternalURL: 'https://__HOSTNAME_EXT__:__WEBSHELL_EXT_SSL_PORT__'
      Websocket:
        ExternalURL: 'wss://__HOSTNAME_EXT__:__WEBSOCKET_EXT_SSL_PORT__/websocket'
        InternalURLs:
          'http://__HOSTNAME_INT__:8005': {}
      Workbench1:
        ExternalURL: 'https://__HOSTNAME_EXT__:__WORKBENCH1_EXT_SSL_PORT__'
      Workbench2:
        ExternalURL: 'https://__HOSTNAME_EXT__:__WORKBENCH2_EXT_SSL_PORT__'
