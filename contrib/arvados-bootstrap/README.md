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
