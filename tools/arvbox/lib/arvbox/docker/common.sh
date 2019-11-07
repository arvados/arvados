# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0


export PATH=${PATH}:/usr/local/go/bin:/var/lib/gems/bin
export GEM_HOME=/var/lib/gems
export GEM_PATH=/var/lib/gems
export npm_config_cache=/var/lib/npm
export npm_config_cache_min=Infinity
export R_LIBS=/var/lib/Rlibs
export HOME=$(getent passwd arvbox | cut -d: -f6)

defaultdev=$(/sbin/ip route|awk '/default/ { print $5 }')
containerip=$(ip addr show $defaultdev | grep 'inet ' | sed 's/ *inet \(.*\)\/.*/\1/')
if test -s /var/run/localip_override ; then
    localip=$(cat /var/run/localip_override)
else
    localip=$containerip
fi

root_cert=/var/lib/arvados/root-cert.pem
root_cert_key=/var/lib/arvados/root-cert.key
server_cert=/var/lib/arvados/server-cert-${localip}.pem
server_cert_key=/var/lib/arvados/server-cert-${localip}.key

declare -A services
services=(
  [workbench]=443
  [workbench2]=3000
  [workbench2-ssl]=3001
  [api]=8004
  [controller]=8003
  [controller-ssl]=8000
  [sso]=8900
  [composer]=4200
  [arv-git-httpd-ssl]=9000
  [arv-git-httpd]=9001
  [keep-web]=9003
  [keep-web-ssl]=9002
  [keepproxy]=25100
  [keepproxy-ssl]=25101
  [keepstore0]=25107
  [keepstore1]=25108
  [ssh]=22
  [doc]=8001
  [websockets]=8005
  [websockets-ssl]=8002
)

if test "$(id arvbox -u 2>/dev/null)" = 0 ; then
    PGUSER=postgres
    PGGROUP=postgres
else
    PGUSER=arvbox
    PGGROUP=arvbox
fi

run_bundler() {
    if test -f Gemfile.lock ; then
        frozen=--frozen
    else
        frozen=""
    fi
    if ! test -x /var/lib/gems/bin/bundler ; then
        bundlergem=$(ls -r $GEM_HOME/cache/bundler-*.gem 2>/dev/null | head -n1 || true)
        if test -n "$bundlergem" ; then
            flock /var/lib/gems/gems.lock gem install --local --no-document $bundlergem
        else
            flock /var/lib/gems/gems.lock gem install --no-document bundler
        fi
    fi
    if ! flock /var/lib/gems/gems.lock bundler install --path $GEM_HOME --local --no-deployment $frozen "$@" ; then
        flock /var/lib/gems/gems.lock bundler install --path $GEM_HOME --no-deployment $frozen "$@"
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
	if ! pip3 install --no-index --find-links /var/lib/pip $1 ; then
            pip3 install $1
	fi
    else
	if ! pip install --no-index --find-links /var/lib/pip $1 ; then
            pip install $1
	fi
    fi
}
