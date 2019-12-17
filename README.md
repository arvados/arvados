[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

[Arvados](https://arvados.org) is a free software distributed computing platform
for bioinformatics, data science, and high throughput analysis of massive data
sets.  Arvados supports a variety of cloud, cluster and HPC environments.

Arvados consists of:

* *Keep*: a petabyte-scale content-addressed distributed storage system for managing and
  storing collections of files, accessible via HTTP and FUSE mount.

* *Crunch*: a Docker-based cluster and HPC workflow engine designed providing
  strong versioning, reproducibilty, and provenance of computations.

* Related services and components including a web workbench for managing files
  and compute jobs, REST APIs, SDKs, and other tools.

## Quick start

To try out Arvados on your local workstation, you can use Arvbox, which
provides Arvados components pre-installed in a Docker container (requires
Docker 1.9+).  After cloning the Arvados git repository:

```
$ cd arvados/tools/arvbox/bin
$ ./arvbox start localdemo
```

In this mode you will only be able to connect to Arvbox from the same host.  To
configure Arvbox to be accessible over a network and for other options see
http://doc.arvados.org/install/arvbox.html for details.

## Documentation

Complete documentation, including a User Guide, Installation documentation and
API documentation is available at http://doc.arvados.org/

If you wish to build the Arvados documentation from a local git clone, see
doc/README.textile for instructions.

## Community

[![Join the chat at https://gitter.im/arvados/community](https://badges.gitter.im/arvados/community.svg)](https://gitter.im/arvados/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

The [arvados community channel](https://gitter.im/arvados/community)
channel at [gitter.im](https://gitter.im) is available for live
discussion and support.

The
[Arvados user mailing list](http://lists.arvados.org/mailman/listinfo/arvados)
is a forum for general discussion, questions, and news about Arvados
development.  The
[Arvados developer mailing list](http://lists.arvados.org/mailman/listinfo/arvados-dev)
is a forum for more technical discussion, intended for developers and
contributors to Arvados.

## Development

[![Build Status](https://ci.arvados.org/buildStatus/icon?job=run-tests)](https://ci.arvados.org/job/run-tests/)
[![Go Report Card](https://goreportcard.com/badge/github.com/curoverse/arvados)](https://goreportcard.com/report/github.com/curoverse/arvados)

The Arvados public bug tracker is located at https://dev.arvados.org/projects/arvados/issues

Continuous integration is hosted at https://ci.arvados.org/

Instructions for setting up a development environment and working on specific
components can be found on the
["Hacking Arvados" page of the Arvados wiki](https://dev.arvados.org/projects/arvados/wiki/Hacking).

## Contributing

When making a pull request, please ensure *every git commit message* includes a one-line [Developer Certificate of Origin](https://dev.arvados.org/projects/arvados/wiki/Developer_Certificate_Of_Origin). If you have already made commits without it, fix them with `git commit --amend` or `git rebase`.

## Licensing

Arvados is Free Software.  See COPYING for information about Arvados Free
Software licenses.
