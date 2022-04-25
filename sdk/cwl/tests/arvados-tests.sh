#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

set -e

if ! arv-get d7514270f356df848477718d58308cc4+94 > /dev/null ; then
    arv-put --portable-data-hash testdir/*
fi
if ! arv-get f225e6259bdd63bc7240599648dde9f1+97 > /dev/null ; then
    arv-put --portable-data-hash hg19/*
fi
if ! arv-get 4d8a70b1e63b2aad6984e40e338e2373+69 > /dev/null ; then
    arv-put --portable-data-hash secondaryFiles/hello.txt*
fi
if ! arv-get 20850f01122e860fb878758ac1320877+71 > /dev/null ; then
    arv-put --portable-data-hash samples/sample1_S01_R1_001.fastq.gz
fi

arvados-cwl-runner 18888-download_def.cwl --scripts scripts/

exec cwltest --test arvados-tests.yml --tool arvados-cwl-runner $@ -- --disable-reuse --compute-checksum --api=containers
