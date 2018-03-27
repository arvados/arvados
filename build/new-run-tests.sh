#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

# First make sure to remove any ARVADOS_ variables from the calling
# environment that could interfere with the tests.
unset $(env | cut -d= -f1 | grep \^ARVADOS_)

# Reset other variables that could affect our [tests'] behavior by
# accident.
GITDIR=
GOPATH=
VENVDIR=
VENV3DIR=
PYTHONPATH=
GEMHOME=
PERLINSTALLBASE=
R_LIBS=

short=
temp=
temp_preserve=

if [[ -z "$WORKSPACE" ]] ; then
    WORKSPACE=$(readlink -f $(dirname $0)/..)
fi

declare -a include_tests
declare -A exclude_tests

declare -A include_install
declare -A exclude_install

. test-library.sh

if [[ $(whoami) = 'arvbox' && -f /usr/local/lib/arvbox/common.sh ]] ; then
    . /usr/local/lib/arvbox/common.sh
fi

interrupt() {
    failures+=("($(basename $0) interrupted)")
    exit_cleanly
}
trap interrupt INT

echo "WORKSPACE is $WORKSPACE"

while [[ -n "$1" ]]
do
    arg="$1"; shift
    case "$arg" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --temp)
            temp="$1"; shift
            temp_preserve=1
            ;;
        --leave-temp)
            temp_preserve=1
            ;;
        --repeat)
            repeat=$((${1}+0)); shift
            ;;
        --retry)
            retry=1
            ;;
        -i)
            include_tests+="$1"; shift
            ;;
        -x)
	    exclude_tests[$1]=1; shift
	    ;;
	--only-install)
	    include_install[$1]=1; shift
	    ;;
	--skip-install)
	    if [[ $1 = all ]] ; then
		include_install["none"]=1
	    else
		exclude_install[$1]=1
	    fi
	    shift
            ;;
	--debug)
	    set -x
	    ;;
        *)
            echo >&2 "$0: Unrecognized option: '$arg'. Try: $0 --help"
            exit 1
            ;;
    esac
done

echo "temp is $temp"

cd $WORKSPACE
find . -name '*.pyc' -delete

if [[ -z "${include_tests[@]}" ]] ; then
    find_run_tests=$(find . -name .run-tests)

    for t in $find_run_tests ; do
	[[ $t =~ ./(.*)/.run-tests ]]
	include_tests+=(${BASH_REMATCH[1]})
    done
fi

for t in "${include_tests[@]}" ; do
    if [[ -n "${exclude_tests[$t]}" ]] ; then
	continue
    fi

    TESTDEPS=()
    TESTS=()
    . $WORKSPACE/$t/.run-tests

    if [[ -n "${TESTDEPS}" ]] ; then
	title "Begin $t"
    fi
    installfail=""
    for TESTDEP in ${TESTDEPS} ; do
	$TESTDEP
	if [[ $? != 0 ]] ; then
	    installfail=1
	    break
	fi
    done
    if [[ -n "$installfail" ]] ; then
	continue
    fi
    for TESTFN in ${TESTS} ; do
	do_test $t $TESTFN
    done
done

exit_cleanly
