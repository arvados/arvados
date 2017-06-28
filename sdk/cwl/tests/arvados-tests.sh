#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

if ! arv-get d7514270f356df848477718d58308cc4+94 > /dev/null ; then
    arv-put --portable-data-hash testdir
fi
exec cwltest --test arvados-tests.yml --tool $PWD/runner.sh $@
