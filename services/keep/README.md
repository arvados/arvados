# Keep

## Building

Install go. It needs at least a more recent version than 1.1.2, available in
Ubuntu packages, to have the crypt/md5 library. More instructions are on the wiki:

https://arvados.org/projects/arvados/wiki/Hacking_Keep

To build do:

    ./go.sh install keep

Keep will be available in `bin/keep`.

There is also an apt repo at apt.arvados.org:

    apt.arvados.org
    deb http://apt.arvados.org/ wheezy main
    apt-get install keep

## Running

    keep -listen=":25107" -volumes="tmp/keep"

For usage via the command line tool or langauge SDKs, you also need the REST
API server and single sign on tools. A build for running all of these inside
docker containers is in the top level `docker` directory in the Arvados source tree.
