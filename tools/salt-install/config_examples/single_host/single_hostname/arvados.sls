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
      name: arvados
      host: 127.0.0.1
      password: changeme_arvados
      user: arvados
      encoding: en_US.utf8
      client_encoding: UTF8

    tls:
      # certificate: ''
      # key: ''
      # required to test with arvados-snakeoil certs
      insecure: true

    ### TOKENS
    tokens:
      system_root: changemesystemroottoken
      management: changememanagementtoken
      rails_secret: changemerailssecrettoken
      anonymous_user: changemeanonymoususertoken

    ### KEYS
    secrets:
      blob_signing_key: changemeblobsigningkey
      workbench_secret_key: changemeworkbenchsecretkey
      dispatcher_access_key: changemedispatcheraccesskey
      dispatcher_secret_key: changeme_dispatchersecretkey
      keep_access_key: changemekeepaccesskey
      keep_secret_key: changemekeepsecretkey

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
          'http://__HOSTNAME__:25107':
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
        ExternalURL: 'https://__HOSTNAME__:__CONTROLLER_EXT_SSL_PORT__'
        InternalURLs:
          'http://controller.internal:8003': {}
      DispatchCloud:
        InternalURLs:
          'http://__HOSTNAME__:9006': {}
      Keepbalance:
        InternalURLs:
          'http://__HOSTNAME__:9005': {}
      Keepproxy:
        ExternalURL: 'https://__HOSTNAME__:__KEEP_EXT_SSL_PORT__'
        InternalURLs:
          'http://keep.internal:25100': {}
      Keepstore:
        InternalURLs:
          'http://keep0.internal:25107': {}
      RailsAPI:
        InternalURLs:
          'http://api.internal:8004': {}
      WebDAV:
        ExternalURL: 'https://__HOSTNAME__:__KEEPWEB_EXT_SSL_PORT__'
        InternalURLs:
          'http://collections.internal:9002': {}
      WebDAVDownload:
        ExternalURL: 'https://__HOSTNAME__:__KEEPWEB_EXT_SSL_PORT__'
      WebShell:
        ExternalURL: 'https://__HOSTNAME__:__WEBSHELL_EXT_SSL_PORT__'
      Websocket:
        ExternalURL: 'wss://__HOSTNAME__:__WEBSOCKET_EXT_SSL_PORT__/websocket'
        InternalURLs:
          'http://ws.internal:8005': {}
      Workbench1:
        ExternalURL: 'https://__HOSTNAME__:__WORKBENCH1_EXT_SSL_PORT__'
      Workbench2:
        ExternalURL: 'https://__HOSTNAME__:__WORKBENCH2_EXT_SSL_PORT__'
