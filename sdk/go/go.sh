#! /bin/sh

rootdir=$(dirname $0)
GOPATH=$rootdir:$GOPATH
export GOPATH

mkdir -p $rootdir/pkg
mkdir -p $rootdir/bin

go get gopkg.in/check.v1

go $*
