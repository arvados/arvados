#! /bin/sh

rootdir=$(dirname $0)
GOPATH=$rootdir:$rootdir/../../sdk/go:$GOPATH
export GOPATH

mkdir -p pkg
mkdir -p bin

go get github.com/gorilla/mux

go $*
