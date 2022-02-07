# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

export RUBY_VERSION=2.7.0
export BUNDLER_VERSION=2.2.19

export DEBIAN_FRONTEND=noninteractive
export PATH=${PATH}:/usr/local/go/bin:/var/lib/arvados/bin
export npm_config_cache=/var/lib/npm
export npm_config_cache_min=Infinity
export R_LIBS=/var/lib/Rlibs
export HOME=$(getent passwd arvbox | cut -d: -f6)
export ARVADOS_CONTAINER_PATH=/var/lib/arvados-arvbox

defaultdev=$(/sbin/ip route|awk '/default/ { print $5 }')
dockerip=$(/sbin/ip route | grep default | awk '{ print $3 }')
containerip=$(ip addr show $defaultdev | grep 'inet ' | sed 's/ *inet \(.*\)\/.*/\1/')
if test -s /var/run/localip_override ; then
    localip=$(cat /var/run/localip_override)
else
    localip=$containerip
fi

root_cert=$ARVADOS_CONTAINER_PATH/root-cert.pem
root_cert_key=$ARVADOS_CONTAINER_PATH/root-cert.key
server_cert=$ARVADOS_CONTAINER_PATH/server-cert-${localip}.pem
server_cert_key=$ARVADOS_CONTAINER_PATH/server-cert-${localip}.key

declare -A services
services=(
  [workbench]=443
  [workbench2]=3000
  [workbench2-ssl]=3001
  [api]=8004
  [controller]=8003
  [controller-ssl]=8000
  [composer]=4200
  [arv-git-httpd-ssl]=9000
  [arv-git-httpd]=9001
  [keep-web]=9003
  [keep-web-ssl]=9002
  [keep-web-dl-ssl]=9004
  [keepproxy]=25100
  [keepproxy-ssl]=25101
  [keepstore0]=25107
  [keepstore1]=25108
  [ssh]=22
  [doc]=8001
  [websockets]=8005
  [websockets-ssl]=8002
  [webshell]=4201
  [webshell-ssl]=4202
)

if test "$(id arvbox -u 2>/dev/null)" = 0 ; then
    PGUSER=postgres
    PGGROUP=postgres
else
    PGUSER=arvbox
    PGGROUP=arvbox
fi

run_bundler() {
    /var/lib/arvados/bin/gem install --no-document bundler:$BUNDLER_VERSION
    if test -f Gemfile.lock ; then
        frozen=--frozen
    else
        frozen=""
    fi
    BUNDLER=bundle
    if test -x $PWD/bin/bundle ; then
	# If present, use the one associated with rails workbench or API
	BUNDLER=$PWD/bin/bundle
    fi
    if ! $BUNDLER install --verbose --local --no-deployment $frozen "$@" ; then
        $BUNDLER install --verbose --no-deployment $frozen "$@"
    fi
}

PYCMD=""
pip_install() {
    pushd /var/lib/pip
    for p in $(ls http*.tar.gz) $(ls http*.tar.bz2) $(ls http*.whl) $(ls http*.zip) ; do
        if test -f $p ; then
            ln -sf $p $(echo $p | sed 's/.*%2F\(.*\)/\1/')
        fi
    done
    popd

    if [ "$PYCMD" = "python3" ]; then
        if ! pip3 install --prefix /usr/local --no-index --find-links /var/lib/pip $1 ; then
            pip3 install --prefix /usr/local $1
        fi
    else
        if ! pip install --no-index --find-links /var/lib/pip $1 ; then
            pip install $1
        fi
    fi
}
