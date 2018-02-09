# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0


export PATH=${PATH}:/usr/local/go/bin:/var/lib/gems/bin
export GEM_HOME=/var/lib/gems
export GEM_PATH=/var/lib/gems
export npm_config_cache=/var/lib/npm
export npm_config_cache_min=Infinity
export R_LIBS=/var/lib/Rlibs

if test -s /var/run/localip_override ; then
    localip=$(cat /var/run/localip_override)
else
    defaultdev=$(/sbin/ip route|awk '/default/ { print $5 }')
    localip=$(ip addr show $defaultdev | grep 'inet ' | sed 's/ *inet \(.*\)\/.*/\1/')
fi

declare -A services
services=(
  [workbench]=80
  [api]=8000
  [sso]=8900
  [composer]=4200
  [arv-git-httpd]=9001
  [keep-web]=9002
  [keepproxy]=25100
  [keepstore0]=25107
  [keepstore1]=25108
  [ssh]=22
  [doc]=8001
  [websockets]=8002
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

pip_install() {
    pushd /var/lib/pip
    for p in $(ls http*.tar.gz) $(ls http*.tar.bz2) $(ls http*.whl) $(ls http*.zip) ; do
        if test -f $p ; then
            ln -sf $p $(echo $p | sed 's/.*%2F\(.*\)/\1/')
        fi
    done
    popd

    if ! pip install --no-index --find-links /var/lib/pip $1 ; then
        pip install $1
    fi
}
