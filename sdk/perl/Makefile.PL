#! /usr/bin/perl
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

use strict;

use ExtUtils::MakeMaker;

WriteMakefile(
    NAME            => 'Arvados',
    VERSION_FROM    => 'lib/Arvados.pm',
    PREREQ_PM       => {
        'JSON'     => 0,
        'LWP'      => 0,
        'Net::SSL' => 0,
    },
);
