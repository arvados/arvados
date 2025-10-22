# Arvados Bootstrap Tools

<!--
Copyright (C) The Arvados Authors. All rights reserved.

SPDX-License-Identifier: Apache-2.0
-->

## Introduction

This package provides scripts to initialize an Arvados cluster with data, built on top of the Python SDK. From inside this directory, you can install it by running:

      pipx install .

or if you're managing your own virtualenvs and have one activated:

      pip install .

## arv-export

`arv-export` saves records from a running Arvados cluster to the directory where you run it. It finds Arvados credentials the same way arv-copy does, by reading `~/.config/arvados/ZZZZZ.conf`, where `ZZZZZ` is a five-alphanumeric cluster ID.

`cd` to a directory where you want to save data and run:

      arv-export [other options] OBJECT_UUID

This will subdirectories inside the current directory with data from Arvados. You can load this data with `arv-import` as described in the next section.

## arv-import

`arv-import` creates records on an Arvados cluster from records previously saved by `arv-export`. It finds Arvados credentials the same way arv-copy does, by reading `~/.config/arvados/ZZZZZ.conf`, where `ZZZZZ` is a five-alphanumeric cluster ID.

`cd` to a directory where you previously saved data with `arv-export` and run:

      arv-import [--project-uuid=UUID] [--no-block-copy] [other options] OBJECT_UUID

`OBJECT_UUID` should match a UUID you exported with `arv-export`.

### Using --no-block-copy

If you have administrator access to the destination cluster, then you have the option to write Keep blocks directly to the underlying storage and skip the normal upload using the `--no-block-copy` option. This is normally faster than uploading the blocks via HTTP, but you are entirely responsible for the separate data transfer. For example, if you use a standard filesystem-backed Keep volume, you might run:

      rsync -r arv-export-data/keep/ root@keep.xurid.example:/var/lib/arvados/keep-data/

The exact process will vary by Keep volume and system configuration. Documenting all the possibilities is outside the scope of this document.

## arv-seed

### Synopsis

arv-seed is a script to bulk create Arvados objects from JSON files.

        arv-seed [options] DIRECTORY [directory ...]

### Configuration

By default, when running as root, this tool will read the cluster
configuration file `$ARVADOS_CONFIG` (default `/etc/arvados/config.yml`),
search for exactly one cluster configuration with a `Controller` endpoint and
`SystemRootToken` configured, and use that.

When running as a non-root user, this tool will search for user credentials
the same way as other Arvados command-line tools.

You can control how to load credentials using the `--client-from` option.

### Input

Each directory will be scanned for files named `NAME.TYPE.json`. `NAME` is any
name you like. `TYPE` is the name of an Arvados API resource type, like `group`,
`collection`, or `container_request`. `TYPE` can be spelled with any
punctuation, use CamelCase or not, and be singular or plural.

Input can be further controlled with "base" JSON that sets attributes for all
objects as well as additional parameters for the Arvados create method. Refer
to the `--help` output for `--base-object` and `--parameters` for details.

### Output

When finished, the tool writes JSON output like this to stdout:

      {
        "created": {"/path1": {… Arvados object…}, …},
        "failed": {"/path2": "error message", …}
      }

For both `created` and `failed`, each key is the absolute path of a JSON file
that the tool read. For `created`, each value is the object that Arvados
returned after creation. For `failed`, each value is an error message that
describes why no object could be created.

### Logging

The tool always logs to syslog. It also logs to stderr if `$TERM` is set.
Control what gets logged with the `--loglevel` option.

### Exit codes

arv-seed uses the following exit codes:

* 0: Created all objects successfully (at least one)
* 1: Early internal error
* 2: Incorrect command line arguments
* 11: Created no objects successfully (at least one)
* 12: Mixed results: some objects were created, others failed
* 66: Did not find any JSON input files (`EX_NOINPUT`)
* 70: Internal error (`EX_SOFTWARE`)
* 78: Could not initialize from configuration (`EX_CONFIG`)

### Example

Read JSON from `~/arv-seed` and create them all in the given directory:

      arv-seed --base='{"owner_uuid":"zzzzz-j7d0g-12345abcde67890"}' ~/arv-seed

### systemd service example

      [Unit]
      After=arvados-railsapi.service arvados-controller.service network-online.target

      [Service]
      Type=oneshot
      StandardOutput=file:%t/%N.json
      ExecStart=/opt/arvados-bootstrap/bin/arv-seed /usr/local/share/arv-seed

## arv-federation-migrate

### Introduction

When using multiple Arvados clusters before a federation, a user would have to create a separate account on each cluster.  Unfortunately, because each account represents a separate "identity", in this system permissions granted to a user on one cluster do not transfer to another cluster, even if the accounts are associated with the same user.

To address this, Arvados supports "federated user accounts".  A federated user account is associated with a specific "home" cluster, and can be used to access other clusters in the federation that trust the home cluster.  When a user arrives at another cluster's Workbench, they select and log in to their home cluster, and then are returned to the starting cluster logged in with the federated user account.

When setting up federation capabilities on existing clusters, some users might already have accounts on multiple clusters.  In order to have a single federated identity, users should be assigned a "home" cluster, and accounts associated with that user on the other (non-home) clusters should be migrated to the new federated user account.  The @arv-federation-migrate@ tool assists with this.

This tool is designed to help an administrator who has access to all clusters in a federation to migrate users who have multiple accounts to a single federated account.

As part of migrating a user, any data or permissions associated with old user accounts will be reassigned to the federated account.

### Step 1: Get a user report

#### With a LoginCluster

When using centralized user database as specified by `LoginCluster` in the config file.

Set the `ARVADOS_API_HOST` and `ARVADOS_API_TOKEN` environment variables to be an admin user on cluster in `LoginCluster` .  It will automatically determine the other clusters that are listed in the federation.

Next, run `arv-federation-migrate` with the `--report` flag:

    $ arv-federation-migrate --report users.csv
    Getting user list from x6b1s
    Getting user list from x3982
    Wrote users.csv

#### Without a LoginCluster

The first step is to create `tokens.csv` and list each cluster and API token to access the cluster.  API tokens must be trusted tokens with administrator access.  This is a simple comma separated value file and can be created in a text editor.  Example:

    x3982.arvadosapi.com,v2/x3982-gj3su-sb6meh2jf145s7x/98d40d70d8862e33d7398213435d1a71a96cf870
    x6b1s.arvadosapi.com,v2/x6b1s-gj3su-dxc87btfv5kg91z/5575d980d3ff6231bb0c692281c42a7541c59417

Next, run `arv-federation-migrate` with the `--tokens` and `--report` flags:

    $ arv-federation-migrate --tokens tokens.csv --report users.csv
    Reading tokens.csv
    Getting user list from x6b1s
    Getting user list from x3982
    Wrote users.csv

### Step 2: Update the user report

This will produce a report of users across all clusters listed in `tokens.csv`, sorted by email address.  This file can be loaded into a text editor or spreadsheet program for ease of viewing and editing.

    email,username,user uuid,primary cluster/user
    person_a@example.com,person_a,x6b1s-tpzed-hb5n7doogwhk6cf,x6b1s
    person_b@example.com,person_b,x3982-tpzed-1vl3k7knf7qihbe,
    person_b@example.com,person_b,x6b1s-tpzed-w4nhkx2rmrhlr54,

The fourth column describes that user's home cluster.  If a user only has one account (identified by email address), the column will be filled in and there is nothing to do.  If the column is blank, that means there is more than one Arvados account associated with the user.  Edit the file and provide the desired home cluster for each user as necessary (note: if there is a LoginCluster, all users will be migrated to the LoginCluster).  It is also possible to change the desired username for a user.  In this example, `person_b@example.com` is assigned the home cluster `x3982`.

    email,username,user uuid,primary cluster/user
    person_a@example.com,person_a,x6b1s-tpzed-hb5n7doogwhk6cf,x6b1s
    person_b@example.com,person_b,x3982-tpzed-1vl3k7knf7qihbe,x3982
    person_b@example.com,person_b,x6b1s-tpzed-w4nhkx2rmrhlr54,x3982

### Step 3: Migrate users

To avoid disruption, advise users to log out and avoid running workflows while performing the migration.

After updating `users.csv`, you can preview the migration using the `--dry-run` option (add `--tokens tokens.csv` if not using LoginCluster).  This will print out what actions the migration will take (as if it were happening) and report possible problems, but not make any actual changes on any cluster:

    $ arv-federation-migrate --dry-run users.csv
    (person_b@example.com) Migrating x6b1s-tpzed-w4nhkx2rmrhlr54 to x3982-tpzed-1vl3k7knf7qihbe

Execute the migration using the `--migrate` option (add `--tokens tokens.csv` if not using LoginCluster):

    $ arv-federation-migrate --migrate users.csv
    (person_b@example.com) Migrating x6b1s-tpzed-w4nhkx2rmrhlr54 to x3982-tpzed-1vl3k7knf7qihbe

After migration, users should select their home cluster when logging into Arvados Workbench.  If a user attempts to log into a migrated user account, they will be redirected to log in with their home cluster.
