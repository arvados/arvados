[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

[![Join the chat at https://gitter.im/arvados/community](https://badges.gitter.im/arvados/community.svg)](https://gitter.im/arvados/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) | [Installing Arvados](https://doc.arvados.org/install/index.html) | [Installing Client SDKs](https://doc.arvados.org/sdk/index.html) | [Report a bug](https://dev.arvados.org/projects/arvados/issues/new) | [Development and Contributing](CONTRIBUTING.md)

[Arvados](https://arvados.org) is a free software distributed computing platform
for bioinformatics, data science, and high throughput analysis of massive data
sets.  Arvados supports a variety of cloud, cluster and HPC environments.

Arvados consists of:

* *Keep*: a petabyte-scale content-addressed distributed storage system for managing and
  storing collections of files, accessible via a variety of methods including
  Arvados APIs, WebDAV, and FUSE file system mount.

* *Crunch*: a Docker-based cloud and HPC workflow engine designed providing
  strong versioning, reproducibilty, and provenance of large-scale computations.

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

Complete documentation, including the [User Guide](https://doc.arvados.org/user/index.html), [Installation documentation](https://doc.arvados.org/install/index.html) and
[API documentation](https://doc.arvados.org/api/index.html) is available at http://doc.arvados.org/

If you wish to build the Arvados documentation from a local git clone, see
doc/README.textile for instructions.

## Community

[![Join the chat at https://gitter.im/arvados/community](https://badges.gitter.im/arvados/community.svg)](https://gitter.im/arvados/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

The [Arvados community channel](https://gitter.im/arvados/community)
channel at [gitter.im](https://gitter.im) is available for live
discussion and support.

The [Arvados developement channel](https://gitter.im/arvados/development)
channel at [gitter.im](https://gitter.im) is used to coordinate development.

The [Arvados user mailing list](http://lists.arvados.org/mailman/listinfo/arvados)
is used to announce new versions and other news.

## Reporting bugs

[Report a bug](https://dev.arvados.org/projects/arvados/issues/new) on the [dev.arvados.org Redmine site](https://dev.arvados.org).

## Development and Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for information about Arvados development and how to contribute to the Arvados project.

## Licensing

Arvados is Free Software.  See COPYING for information about Arvados Free
Software licenses.
