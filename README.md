Welcome to Arvados
==================

[Arvados](https://arvados.org) is a free software distributed computing platform
for bioinformatics, data science, and high throughput analysis of massive data
sets.  Arvados supports a variety of cloud, cluster and HPC environments.

Arvados consists of:

* *Keep*: a petabyte-scale content-addressed distributed storage system for managing and
  storing collections of files, accessible via HTTP and FUSE mount.

* *Crunch*: a Docker-based workflow engine designed providing strong
  versioning, reproducibilty, and provenance of computations.

* Related services and components including a web workbench for managing files
  and compute jobs, REST APIs, SDKs, and other tools.

## Quick start

To try out Arvados quickly, you can use Arvbox, which provides Arvados
components pre-installed in a Docker container (requires Docker 1.9+).  After
cloning the Arvados git repository:

```
$ cd arvados/tools/arvbox/bin
$ ./arvbox start localdemo
```

See http://doc.arvados.org/install/arvbox.html for details and documentation.

## Documentation

Complete documentation, including a User Guide, Installation documentation and
API documentation is available at http://doc.arvados.org/

If you wish to build the Arvados documentation from a local git clone, see
doc/README.textile for instructions.

## Community

The [#arvados](irc://irc.oftc.net:6667/#arvados IRC) (Internet Relay Chat)
channel at the
[Open and Free Technology Community (irc.oftc.net)](http://www.oftc.net/oftc/)
is available for live discussion and support.  You can use a traditional IRC
client or [join OFTC over the web.](https://webchat.oftc.net/?channels=arvados)

The
[Arvados user mailing list](http://lists.arvados.org/mailman/listinfo/arvados)
is a forum for general discussion, questions, and news about Arvados
development.  The
[Arvados developer mailing list](http://lists.arvados.org/mailman/listinfo/arvados-dev)
is a forum for more technical discussion, intended for developers and
contributers to Arvados.

## Development

[![Build Status](https://ci.curoverse.com/buildStatus/icon?job=arvados-api-server)](https://ci.curoverse.com/job/arvados-api-server/)

The Arvados public bug tracker is located at https://dev.arvados.org/projects/arvados/issues

Continuous integration is hosted at https://ci.curoverse.com/

## Licensing

Arvados is Free Software.  See COPYING for information about Arvados Free
Software licenses.
