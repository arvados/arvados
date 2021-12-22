#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# When run with WORKSPACE pointing at a git checkout of arvados, this script
# calculates the package version of an Arvados component.

# set to --no-cache-dir to disable pip caching
CACHE_FLAG=
STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
DASHQ_UNLESS_DEBUG=-q
ITERATION="${ARVADOS_BUILDING_ITERATION:-1}"

. `dirname "$(readlink -f "$0")"`/run-library.sh

TYPE_LANG=$1
SRC_PATH=$2

if [[ "$TYPE_LANG" == "" ]] || [[ "$SRC_PATH" == "" ]]; then
  echo "Syntax: $0 <lang> <src_path>"
  echo
  echo "Example: $0 go cmd/arvados-client"
  echo "Example: $0 python3 services/fuse"
  echo
  exit 1
fi

if [[ "$WORKSPACE" == "" ]]; then
  echo "The WORKSPACE environment variable must be set, pointing at the root of the arvados git tree"
  exit 1
fi


debug_echo "package_go_binary $SRC_PATH"

if [[ "$TYPE_LANG" == "go" ]]; then
  calculate_go_package_version go_package_version $SRC_PATH
  echo "${go_package_version}-${ITERATION}"
elif [[ "$TYPE_LANG" == "python3" ]]; then

  cd $WORKSPACE/$SRC_PATH

  rm -rf dist/*

  # Get the latest setuptools
  if ! pip3 install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U 'setuptools<45'; then
    echo "Error, unable to upgrade setuptools with"
    echo "  pip3 install $DASHQ_UNLESS_DEBUG $CACHE_FLAG -U 'setuptools<45'"
    exit 1
  fi
  # filter a useless warning (when building the cwltest package) from the stderr output
  if ! python3 setup.py $DASHQ_UNLESS_DEBUG sdist 2> >(grep -v 'warning: no previously-included files matching' |grep -v 'for version number calculation'); then
    echo "Error, unable to run python3 setup.py sdist for $SRC_PATH"
    exit 1
  fi

  PYTHON_VERSION=$(awk '($1 == "Version:"){print $2}' *.egg-info/PKG-INFO)
  UNFILTERED_PYTHON_VERSION=$(echo -n $PYTHON_VERSION | sed s/\.dev/~dev/g |sed 's/\([0-9]\)rc/\1~rc/g')

  echo "${UNFILTERED_PYTHON_VERSION}-${ITERATION}"
fi

