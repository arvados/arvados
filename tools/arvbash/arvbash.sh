#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# bash functions for managing Arvados tokens and other conveniences.

read -rd "\000" helpmessage <<EOF
$(basename $0): bash functions for managing Arvados tokens and other shortcuts.

Syntax:
        . $0            # activate for current shell
        $0 --install    # install into .bashrc

arvswitch <name>
  Set ARVADOS_API_HOST and ARVADOS_API_TOKEN in the current environment based on
  $HOME/.config/arvados/<name>.conf
  With no arguments, list available Arvados configurations.

arvsave <name>
  Save values of ARVADOS_API_HOST and ARVADOS_API_TOKEN in the current environment to
  $HOME/.config/arvados/<name>.conf

arvrm <name>
  Delete $HOME/.config/arvados/<name>.conf

arvboxswitch <name>
  Set ARVBOX_CONTAINER to <name>
  With no arguments, list available arvboxes.

arvopen:
  Open an Arvados uuid in web browser (http://arvadosapi.com)

arvissue
  Open an Arvados ticket in web browser (http://dev.arvados.org)

EOF

if [[ "$1" = "--install" ]] ; then
    this=$(readlink -f $0)
    if ! grep ". $this" ~/.bashrc >/dev/null ; then
        echo ". $this" >> ~/.bashrc
        echo "Installed into ~/.bashrc"
    else
        echo "Already installed in ~/.bashrc"
    fi
elif ! [[ $0 =~ bash$ ]] ; then
    echo "$helpmessage"
fi

HISTIGNORE=$HISTIGNORE:'export ARVADOS_API_TOKEN=*'

arvswitch() {
    if [[ -n "$1" ]] ; then
        if [[ -f $HOME/.config/arvados/$1.conf ]] ; then
            unset ARVADOS_API_HOST_INSECURE
            for a in $(cat $HOME/.config/arvados/$1.conf) ; do export $a ; done
            echo "Switched to $1"
        else
            echo "$1 unknown"
        fi
    else
        echo "Switch Arvados environment conf"
        echo "Usage: arvswitch name"
        echo "Available confs:" $((cd $HOME/.config/arvados && ls --indicator-style=none *.conf) | rev | cut -c6- | rev)
    fi
}

arvsave() {
    if [[ -n "$1" ]] ; then
	touch $HOME/.config/arvados/$1.conf
	chmod 0600 $HOME/.config/arvados/$1.conf
        env | grep ARVADOS_ > $HOME/.config/arvados/$1.conf
    else
        echo "Save current Arvados environment variables to conf file"
        echo "Usage: arvsave name"
    fi
}

arvrm() {
    if [[ -n "$1" ]] ; then
        if [[ -f $HOME/.config/arvados/$1.conf ]] ; then
            rm $HOME/.config/arvados/$1.conf
        else
            echo "$1 unknown"
        fi
    else
        echo "Delete Arvados environment conf"
        echo "Usage: arvrm name"
    fi
}

arvboxswitch() {
    if [[ -n "$1" ]] ; then
        if [[ -d $HOME/.arvbox/$1 ]] ; then
            export ARVBOX_CONTAINER=$1
            echo "Arvbox switched to $1"
        else
            echo "$1 unknown"
        fi
    else
        if test -z "$ARVBOX_CONTAINER" ; then
            ARVBOX_CONTAINER=arvbox
        fi
        echo "Switch Arvbox environment conf"
        echo "Usage: arvboxswitch name"
        echo "Your current container is: $ARVBOX_CONTAINER"
        echo "Available confs:" $(cd $HOME/.arvbox && ls --indicator-style=none)
    fi
}

arvopen() {
    if [[ -n "$1" ]] ; then
        xdg-open https://arvadosapi.com/$1
    else
        echo "Open Arvados uuid in browser"
        echo "Usage: arvopen uuid"
    fi
}

arvissue() {
    if [[ -n "$1" ]] ; then
        xdg-open https://dev.arvados.org/issues/$1
    else
        echo "Open Arvados issue in browser"
        echo "Usage: arvissue uuid"
    fi
}
