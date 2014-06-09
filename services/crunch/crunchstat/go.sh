#! /bin/sh

# Wraps the 'go' executable with some environment setup.  Sets GOPATH, creates
# 'pkg' and 'bin' directories, automatically installs dependencies, then runs
# the underlying 'go' executable with any command line parameters provided to
# the script.

rootdir=$(readlink -f $(dirname $0))
GOPATH=$rootdir:$rootdir/../../sdk/go:$GOPATH
export GOPATH

mkdir -p $rootdir/pkg
mkdir -p $rootdir/bin

go $*
