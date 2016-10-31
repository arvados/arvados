
if test -s /var/run/localip_override ; then
    localip=$(cat /var/run/localip_override)
else
    defaultdev=$(/sbin/ip route|awk '/default/ { print $5 }')
    localip=$(ip addr show $defaultdev | grep 'inet ' | sed 's/ *inet \(.*\)\/.*/\1/')
fi

export GEM_HOME=/var/lib/gems
export GEM_PATH=/var/lib/gems

declare -A services
services=(
  [workbench]=80
  [api]=8000
  [sso]=8900
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
    if ! flock /var/lib/gems/gems.lock bundle install --path $GEM_HOME --local --no-deployment $frozen "$@" ; then
        flock /var/lib/gems/gems.lock bundle install --path $GEM_HOME --no-deployment $frozen "$@"
    fi
}

pip_install() {
    pushd /var/lib/pip
    for p in $(ls http*.tar.gz) $(ls http*.whl) $(ls http*.zip) ; do
        if test -f $p ; then
            ln -sf $p $(echo $p | sed 's/.*%2F\(.*\)/\1/')
        fi
    done
    popd

    if ! pip install --no-index --find-links /var/lib/pip $1 ; then
        pip install $1
    fi
}
