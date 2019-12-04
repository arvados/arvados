#!/bin/bash

set -e -o pipefail
commit="$1"
versionglob="[0-9].[0-9]*.[0-9]*"

if ! git describe --exact-match --match "$versionglob" "$commit" 2>/dev/null; then
    if git merge-base --is-ancestor "$commit" origin/master; then
        # x.(y+1).0.preTIMESTAMP, where x.y.z is the newest version that does not contain $commit
        v=$(git tag | grep -vFf <(git tag --contains "$commit") | sort -Vr | head -n1 | perl -pe 's/\.(\d+)\.\d+/".".($1+1).".0"/e')
    else
        # x.y.(z+1).preTIMESTAMP, where x.y.z is the latest released ancestor of $commit
        v=$(git describe --abbrev=0 --match "$versionglob" "$commit" | perl -pe 's/(\d+)$/$1+1/e')
    fi
    ts=$(TZ=UTC git log -n1 --format=%cd --date="format-local:%Y%m%d%H%M%S" "$commit")
    echo "$v.pre$ts"
fi
