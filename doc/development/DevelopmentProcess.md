[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Development Process

This document is intended for core engineers who work on the `main` branch of Arvados.

## Two Remotes

This document assumes you have two remotes, where `origin` refers to `git.arvados.org` and `github` refers to `github.com`:

```sh
$ git remote -v
github	git@github.com:arvados/arvados.git (fetch)
github	git@github.com:arvados/arvados.git (push)
origin	git@git.arvados.org:arvados.git (fetch)
origin	git@git.arvados.org:arvados.git (push)
```

## Fetch a GitHub Pull Request as a Branch

If someone has submitted a pull request, you can create a local branch from it by running:

```sh
$ git fetch github "pull/PRNUM/head:BRANCHNAME"
```

`PRNUM` is the pull request ID number at the end of the GitHub URL. `BRANCHNAME` is any name you want to give it. You're encouraged to follow the Arvados convention of `PRNUM-brief-description`.

## Reviewing a Pull Request

Reviewing a pull request is about verifying that the branch follows all our [coding standards](CodingStandards.md). You should be able to verify that the ready-to-merge checklist is complete and accurate: the branch does what it says, tests pass, it follows our style, etc.

If you notice scale issues, bugs, missing documentation, etc., you can bring that up as part of the review and it should be addressed. However, the *point* of review is *not* to try to find problems. The *point* is to verify that the branch solves a problem and the code is maintainable.

## Merging a Pull Request

When a branch passes review, it should be merged to `main`. Core engineers can (and normally do) merge their own branches. Contributions from others need to be merged by a core engineer. Either way, the process is:

```sh
$ git switch main
$ git pull --ff-only
$ git merge --no-ff BRANCHREF
# Make sure the commit message includes an issue ref and your DCO signoff.
$ git push origin main
```
