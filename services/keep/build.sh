#! /bin/sh

# This script builds a Keep executable and installs it in
# ./bin/keep.
#
# In idiomatic Go style, a user would install Keep with something
# like:
#
#     go get arvados.org/keep
#     go install arvados.org/keep
#
# which would download both the Keep source and any third-party
# packages it depends on.
#
# Since the Keep source is bundled within the overall Arvados source,
# "go get" is not the primary tool for delivering Keep source and this
# process doesn't work.  Instead, this script sets the environment
# properly and fetches any necessary dependencies by hand.

if [ -z "$GOPATH" ]
then
    GOPATH=$(pwd)
else
    GOPATH=$(pwd):${GOPATH}
fi

export GOPATH

set -o errexit   # fail if any command returns an error

mkdir -p pkg
mkdir -p bin
go get github.com/gorilla/mux
go install keep
ls -l bin/keep
echo "success!"
