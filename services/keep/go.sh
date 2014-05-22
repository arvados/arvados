#! /bin/sh

rootdir=$(dirname $0)
GOPATH=$rootdir:$rootdir/../../sdk/go:$GOPATH
export GOPATH

mkdir -p $rootdir/pkg
mkdir -p $rootdir/bin

go get github.com/gorilla/mux

go $*
