#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script installs all the Python packages and Ansible collections necessary
# to run Arvados playbooks. In order to provide the best experience and stay
# maintainable, it follows a couple of rules:
#  1. It is not *too* automatic.
#     If it can't do the thing you ask, it will fail.
#     It will not change system configuration or elevate privileges.
#  2. Running without options does the most recommended thing.
#     You have to specify options to do anything more advanced.

set -e
set -u

ANSIBLE_DIR="$(dirname "$0")"
ANSIBLE_PKG=ansible-core
EX_UNAVAILABLE=69
EX_SOFTWARE=70
EX_CONFIG=78

errexit() {
    local exitcode="$1"; shift
    echo ERROR: "$@"
    exit "$exitcode"
}

usage() {
    if [ $# -eq 0 ]; then
        exitcode=0
        out_fd=1
    else
        echo ERROR: "$@" >&2
        exitcode=2
        out_fd=2
    fi
    cat >&"$out_fd" <<EOF
usage: install-ansible.sh <-V | VIRTUALENV_DIR>

By default this script installs Ansible with pipx. You can install Ansible to a
virtualenv by naming a directory or using the -V option.

options:
  -V  Install Ansible to the currently activated virtualenv
EOF
    exit "$exitcode"
}

VENVDIR=
while getopts Vh opt; do
    case "$opt" in
        V)
            if ! [ -e "${VIRTUAL_ENV:-/nonexistent}/bin/pip" ]
            then usage "must activate a virtualenv before using -V"
            fi
            ;;
        h) usage ;;
        "?") usage "unknown option \`$OPTARG\`" ;;
    esac
done
shift $((OPTIND - 1))
case "$VENVDIR" in
    "")
        if [ $# -gt 1 ]
        then usage "too many arguments"
        fi
        VENVDIR="${1:-}"
        ;;
    *)
        if [ $# -gt 0 ]
        then usage "cannot specify a virtualenv directory with -V"
        fi
        ;;
esac

case "$VENVDIR" in
    "") # pipx install
        pipx --version >/dev/null ||
            errexit "$EX_UNAVAILABLE" "failed to run pipx"

        ansible_req="$(grep -E "^$ANSIBLE_PKG[^-_[:alnum:]]" "$ANSIBLE_DIR/requirements.txt")" ||
            errexit "$EX_SOFTWARE" "failed to find $ANSIBLE_PKG requirement in requirements.txt"

        pipx install "$ansible_req" ||
            errexit "$?" "failed to pipx install \`$ansible_pkg\`"

        pipx runpip "$ANSIBLE_PKG" install -r "$ANSIBLE_DIR/requirements.txt" ||
            errexit "$?" "failed to install requirements.txt"

        VENVDIR="$(pipx environment --value=PIPX_LOCAL_VENVS)/$ANSIBLE_PKG" ||
            errexit "$EX_CONFIG" "failed to load pipx environment"

        ;;

    *) # pip install inside $VENVDIR
        PIP="$VENVDIR/bin/pip"
        if ! [ -x "$PIP" ]
        then
            python3 -m venv "$VENVDIR" ||
                errexit "$?" "failed to create virtualenv at $VENVDIR"

            if ! [ -x "$PIP" ]
            then errexit "$EX_SOFTWARE" "failed to find pip after creating virtualenv at $VENVDIR"
            fi
        fi

        "$PIP" install -r "$ANSIBLE_DIR/requirements.txt" ||
            errexit "$?" "failed to pip install requirements.txt"

        ;;
esac

"$VENVDIR/bin/ansible-galaxy" install -r "$ANSIBLE_DIR/requirements.yml" ||
    errexit "$?" "failed to install requirements.yml with ansible-galaxy"

printf "%s\n" "" "Ansible successfully installed!"
