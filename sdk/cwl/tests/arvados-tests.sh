#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

if ! arv-get d7514270f356df848477718d58308cc4+94 > /dev/null ; then
    arv-put --portable-data-hash testdir/*
fi
if ! arv-get f225e6259bdd63bc7240599648dde9f1+97 > /dev/null ; then
    arv-put --portable-data-hash hg19/*
fi
exec cwltest --test arvados-tests.yml --tool arvados-cwl-runner $@ -- --disable-reuse --compute-checksum
