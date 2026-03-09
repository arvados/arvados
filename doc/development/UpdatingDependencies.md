[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Updating dependencies

## Go

(see also: [real documentation](https://go.dev/doc/modules/managing-dependencies))

Update a single dependency:

~/arvados\$ go get github.com/docker/docker@latest

Update all dependencies:

~/arvados\$ go get -u -t ./…

Then sync:

~/arvados\$ go mod tidy

This is a good time to review “replace” directives in source:go.mod and find better solutions to issues that are currently handled by pinning modules to old versions or unmaintained forks.
