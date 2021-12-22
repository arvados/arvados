#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail
commit="$1"
versionglob="[0-9].[0-9]*.[0-9]*"
devsuffix="~dev"

# automatically assign version
#
# handles the following cases:
#
# 1. commit is directly tagged.  print that.
#
# 2. commit is on main or a development branch, the nearest tag is older
#    than commit where this branch joins main.
#    -> take greatest version tag in repo X.Y.Z and assign X.(Y+1).0
#
# 3. commit is on a release branch, the nearest tag is newer
#    than the commit where this branch joins main.
#    -> take nearest tag X.Y.Z and assign X.Y.(Z+1)

tagged=$(git tag --points-at "$commit")

if [[ -n "$tagged" ]] ; then
    echo $tagged
else
    # 1. get the nearest tag with 'git describe'
    # 2. get the merge base between this commit and main
    # 3. if the tag is an ancestor of the merge base,
    #    (tag is older than merge base) increment minor version
    #    else, tag is newer than merge base, so increment point version

    nearest_tag=$(git describe --tags --abbrev=0 --match "$versionglob" "$commit")
    merge_base=$(git merge-base origin/main "$commit")

    if git merge-base --is-ancestor "$nearest_tag" "$merge_base" ; then
        # x.(y+1).0~devTIMESTAMP, where x.y.z is the newest version that does not contain $commit
	# grep reads the list of tags (-f) that contain $commit and filters them out (-v)
	# this prevents a newer tag from retroactively changing the versions of everything before it
        v=$(git tag | grep -vFf <(git tag --contains "$commit") | sort -Vr | head -n1 | perl -pe 's/(\d+)\.(\d+)\.\d+.*/"$1.".($2+1).".0"/e')
    else
        # x.y.(z+1)~devTIMESTAMP, where x.y.z is the latest released ancestor of $commit
        v=$(echo $nearest_tag | perl -pe 's/(\d+)$/$1+1/e')
    fi
    isodate=$(TZ=UTC git log -n1 --format=%cd --date=iso "$commit")
    ts=$(TZ=UTC date --date="$isodate" "+%Y%m%d%H%M%S")
    echo "${v}${devsuffix}${ts}"
fi
