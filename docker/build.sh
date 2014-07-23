#! /bin/bash

# make sure Ruby 1.9.3 is installed before proceeding
if ! ruby -e 'exit RUBY_VERSION >= "1.9.3"' 2>/dev/null
then
    echo "Installing Arvados requires at least Ruby 1.9.3."
    echo "You may need to enter your password."
    read -p "Press Ctrl-C to abort, or else press ENTER to install ruby1.9.3 and continue. " unused

    sudo apt-get update
    sudo apt-get -y install ruby1.9.3
fi

function usage {
    echo >&2
    echo >&2 "usage: $0 [options]"
    echo >&2
    echo >&2 "Calling $0 without arguments will build all Arvados docker images"
    echo >&2
    echo >&2 "$0 options:"
    echo >&2 "  -h, --help   Print this help text"
    echo >&2 "  clean        Clear all build information"
    echo >&2 "  realclean    clean and remove all Arvados Docker images except arvados/debian"
    echo >&2 "  deepclean    realclean and remove arvados/debian, crosbymichael/skydns and "
    echo >&2 "               crosbymichael/skydns Docker images"
    echo >&2
}

if [ "$1" = '-h' ] || [ "$1" = '--help' ]; then
  usage
  exit 1
fi

build_tools/build.rb

if [[ "$?" == "0" ]]; then
    DOCKER=`which docker.io`

    if [[ "$DOCKER" == "" ]]; then
      DOCKER=`which docker`
    fi

    DOCKER=$DOCKER /usr/bin/make -f build_tools/Makefile $*
fi
