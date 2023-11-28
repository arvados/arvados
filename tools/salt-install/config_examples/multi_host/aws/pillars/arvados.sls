---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set _workers = ("__CONTROLLER_MAX_WORKERS__" or grains['num_cpus']*2)|int %}
{%- set max_workers = [_workers, 8]|max %}
{%- set max_reqs = ("__CONTROLLER_MAX_QUEUED_REQUESTS__" or 128)|int %}
{%- set database_host = ("__DATABASE_EXTERNAL_SERVICE_HOST_OR_IP__" or "__DATABASE_INT_IP__") %}
{%- set database_name = "__DATABASE_NAME__" %}
{%- set database_user = "__DATABASE_USER__" %}
{%- set database_password = "__DATABASE_PASSWORD__" %}

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
  dispatcher:
    pkg:
      name: arvados-dispatch-cloud
    service:
      name: arvados-dispatch-cloud

  ### ARVADOS CLUSTER CONFIG
  cluster:
    name: __CLUSTER__
    domain: __DOMAIN__

    database:
      # max concurrent connections per arvados server daemon
      # connection_pool_max: 32
      name: {{ database_name }}
      host: {{ database_host }}
      password: {{ database_password }}
      user: {{ database_user }}
      encoding: en_US.utf8
      client_encoding: UTF8

    tls:
      # certificate: ''
      # key: ''
      # required to test with arvados-snakeoil certs
      insecure: false

    resources:
      virtual_machines:
        shell:
          name: shell.__DOMAIN__
          backend: __SHELL_INT_IP__
          port: 4200

    ### TOKENS
    tokens:
      system_root: __SYSTEM_ROOT_TOKEN__
      management: __MANAGEMENT_TOKEN__
      anonymous_user: __ANONYMOUS_USER_TOKEN__

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

    ### API
    API:
      MaxConcurrentRailsRequests: {{ max_workers * 2 }}
      MaxConcurrentRequests: {{ max_reqs }}
      MaxQueuedRequests: {{ max_reqs }}

    ### CONTAINERS
    {%- set dispatcher_ssh_privkey = "__DISPATCHER_SSH_PRIVKEY__" %}
    Containers:
      MaxRetryAttempts: 10
      CloudVMs:
        ResourceTags:
          Name: __CLUSTER__-compute-node
        BootProbeCommand: 'systemctl is-system-running'
        ImageID: __COMPUTE_AMI__
        Driver: ec2
        DriverParameters:
          Region: __COMPUTE_AWS_REGION__
          EBSVolumeType: gp3
          AdminUsername: __COMPUTE_USER__
          ### This SG should allow SSH from the dispatcher to the compute nodes
          SecurityGroupIDs: ['__COMPUTE_SG__']
          SubnetID: __COMPUTE_SUBNET__
          IAMInstanceProfile: __CLUSTER__-compute-node-00-iam-role
      DispatchPrivateKey: {{ dispatcher_ssh_privkey|yaml_dquote }}

    ### VOLUMES
    ## This should usually match all your `keepstore` instances
    Volumes:
      # the volume name will be composed with
      # <cluster>-nyw5e-<volume>
      __CLUSTER__-nyw5e-000000000000000:
        Replication: 2
        Driver: S3
        DriverParameters:
          Bucket: __KEEP_AWS_S3_BUCKET__
          IAMRole: __KEEP_AWS_IAM_ROLE__
          Region: __KEEP_AWS_REGION__
          # IMPORTANT: The default value for PrefixLength is 0, and should not
          # be changed once the volume is in use. For new installations it's
          # recommended to set it to 3 for better performance.
          # See: https://doc.arvados.org/install/configure-s3-object-storage.html
          PrefixLength: 3

    Users:
      NewUsersAreActive: true
      AutoAdminFirstUser: true
      AutoSetupNewUsers: true
      AutoSetupNewUsersWithRepository: true

    Services:
      Controller:
        ExternalURL: 'https://__DOMAIN__:__CONTROLLER_EXT_SSL_PORT__'
        InternalURLs:
          'http://localhost:8003': {}
      DispatchCloud:
        InternalURLs:
          'http://__DISPATCHER_INT_IP__:9006': {}
      Keepbalance:
        InternalURLs:
          'http://__KEEPBALANCE_INT_IP__:9005': {}
      Keepproxy:
        ExternalURL: 'https://keep.__DOMAIN__:__KEEP_EXT_SSL_PORT__'
        InternalURLs:
          'http://localhost:25107': {}
      Keepstore:
        InternalURLs:
          'http://__KEEPSTORE0_INT_IP__:25107': {}
      RailsAPI:
        InternalURLs:
          'http://localhost:8004': {}
      WebDAV:
        ExternalURL: 'https://*.collections.__DOMAIN__:__KEEPWEB_EXT_SSL_PORT__/'
        InternalURLs:
          'http://__KEEPWEB_INT_IP__:9002': {}
      WebDAVDownload:
        ExternalURL: 'https://download.__DOMAIN__:__KEEPWEB_EXT_SSL_PORT__'
      WebShell:
        ExternalURL: 'https://webshell.__DOMAIN__:__KEEPWEB_EXT_SSL_PORT__'
      Websocket:
        ExternalURL: 'wss://ws.__DOMAIN__/websocket'
        InternalURLs:
          'http://localhost:8005': {}
      Workbench1:
        ExternalURL: 'https://workbench.__DOMAIN__:__WORKBENCH1_EXT_SSL_PORT__'
      Workbench2:
        ExternalURL: 'https://workbench2.__DOMAIN__:__WORKBENCH2_EXT_SSL_PORT__'

    InstanceTypes:
      t3small:
        ProviderType: t3.small
        VCPUs: 2
        RAM: 2GiB
        AddedScratch: 50GB
        Price: 0.0208
      c5large:
        ProviderType: c5.large
        VCPUs: 2
        RAM: 4GiB
        AddedScratch: 50GB
        Price: 0.085
      m5large:
        ProviderType: m5.large
        VCPUs: 2
        RAM: 8GiB
        AddedScratch: 50GB
        Price: 0.096
      c5xlarge:
        ProviderType: c5.xlarge
        VCPUs: 4
        RAM: 8GiB
        AddedScratch: 100GB
        Price: 0.17
      m5xlarge:
        ProviderType: m5.xlarge
        VCPUs: 4
        RAM: 16GiB
        AddedScratch: 100GB
        Price: 0.192
      m5xlarge_extradisk:
        ProviderType: m5.xlarge
        VCPUs: 4
        RAM: 16GiB
        AddedScratch: 400GB
        Price: 0.193
      c52xlarge:
        ProviderType: c5.2xlarge
        VCPUs: 8
        RAM: 16GiB
        AddedScratch: 200GB
        Price: 0.34
      m52xlarge:
        ProviderType: m5.2xlarge
        VCPUs: 8
        RAM: 32GiB
        AddedScratch: 200GB
        Price: 0.384
      c54xlarge:
        ProviderType: c5.4xlarge
        VCPUs: 16
        RAM: 32GiB
        AddedScratch: 400GB
        Price: 0.68
      m54xlarge:
        ProviderType: m5.4xlarge
        VCPUs: 16
        RAM: 64GiB
        AddedScratch: 400GB
        Price: 0.768
