[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Updating dependencies

## Go

(See also: [Golang documentation](https://go.dev/doc/modules/managing-dependencies))

Update a single dependency:

```sh
~/arvados$ go get github.com/docker/docker@latest
```

Update all dependencies:

```sh
~/arvados$ go get -u -t ./…
```

Then sync:

```sh
~/arvados$ go mod tidy
```

This is a good time to review “replace” directives in source:go.mod and find better solutions to issues that are currently handled by pinning modules to old versions or unmaintained forks.
