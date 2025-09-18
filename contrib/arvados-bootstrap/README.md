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
