#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This in an additional integration test suite for Arvados specific
# bugs and features that are not covered by the unit tests or CWL
# conformance tests.
#

set -ex

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

# Use the python executor associated with the installed OS package, if present.
python=$(((ls /usr/share/python3*/dist/python3-arvados-cwl-runner/bin/python || echo python3) | head -n1) 2>/dev/null)

# Test for #18888
# This is a standalone test because the bug was observed with this
# command line and was thought to be due to command line handling.
arvados-cwl-runner 18888-download_def.cwl --scripts scripts/

# Test for #19070
# The most effective way to test this seemed to be to write an
# integration test to check for the expected behavior.
$python test_copy_deps.py

# Test for #17004
# Checks that the final output collection has the expected properties.
python test_set_output_prop.py

# Run integration tests
exec cwltest --test arvados-tests.yml --tool arvados-cwl-runner $@ -- --disable-reuse --compute-checksum --api=containers
