#! /usr/bin/perl

use strict;

use ExtUtils::MakeMaker;

WriteMakefile(
    NAME            => 'Arvados',
    VERSION_FROM    => 'lib/Arvados.pm'
);
