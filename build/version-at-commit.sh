#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail
commit="$1"
devsuffix="~dev"

# automatically assign *development* version
#
# handles the following cases:
#
# *  commit is on main or a development branch, the nearest tag is older
#    than commit where this branch joins main.
#    -> take greatest version tag in repo X.Y.Z and assign X.(Y+1).0
#
# *  commit is on a release branch, the nearest tag is newer
#    than the commit where this branch joins main.
#    -> take nearest tag X.Y.Z and assign X.Y.(Z+1)

# X.Y.Z releases where Z=0 are called major
# releases and X.Y.Z releases where Z>1 are called point releases.
#
# The development process distinction is that X.Y.0 releases are
# branched from main and then subsequent X.Y.Z releases cherry-pick
# individual features from main onto the "X.Y-staging" branch.
#
# In semantic versioning terminology an "X.Y.0" release which only
# increments Y is called a "minor" release but typically these
# releases have significant changes that calling them "minor" in
# communications with users feels misleading.
#
# Incrementing X is reserved for times when a release has significant
# backwards-incompatible changes, which we don't do very often and try
# to avoid.
#
# In order to assign a useful development version, we need to
# determine if we're on the main branch (or a development branch off
# main) or on a release branch.  We do this by looking at the point
# where the current commit history branched from main.
#
# If a new major version appeared on a branch (not directly in the
# history of main), the merge-base between main and the release should
# be tagged as "development-X.Y.Z" so that version-at-commit can
# figure out what to do.

# 1. get the nearest tag with 'git describe'
# 2. get the merge base between this commit and main
# 3. if the tag is an ancestor of the merge base,
#    (tag is older than merge base) increment minor version
#    else, tag is newer than merge base, so increment point version

nearest_tag=$(git describe --abbrev=0 "$commit")
merge_base=$(git merge-base origin/main "$commit")

if git merge-base --is-ancestor "$nearest_tag" "$merge_base" ; then
    # the nearest tag appears before the merge base with main (the
    # branch point), so assume this is a tag for the previous major
    # release (or a tag with the "development-" prefix indicating the
    # point where a major release branched off).  Subsequent
    # development versions are given the anticipated version for the
    # next major release.
    #
    # x.(y+1).0~devTIMESTAMP, where x.y.z is the newest version that does not contain $commit
    # grep reads the list of tags (-f) that contain $commit and filters them out (-v)
    # this prevents a newer tag from retroactively changing the versions of everything before it
    v=$(git tag | grep -vFf <(git tag --contains "$merge_base") | sort -Vr | head -n1 | perl -pe 's/(\d+)\.(\d+)\.\d+.*/"$1.".($2+1).".0"/e')
else
    # the nearest tag comes after the merge base with main (the branch
    # point).  Assume this means this is a point release branch,
    # following a major release.
    #
    # x.y.(z+1)~devTIMESTAMP, where x.y.z is the latest released ancestor of $commit
    v=$(echo $nearest_tag | perl -pe 's/(\d+)$/$1+1/e')
fi

# strip the "development-" prefix
v=$(echo $v | perl -pe 's/^development-//')

isodate=$(TZ=UTC git log -n1 --format=%cd --date=iso "$commit")
ts=$(TZ=UTC date --date="$isodate" "+%Y%m%d%H%M%S")
echo "${v}${devsuffix}${ts}"
