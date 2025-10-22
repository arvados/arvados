[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

[![Join the chat at https://gitter.im/arvados/community](https://badges.gitter.im/arvados/community.svg)](https://gitter.im/arvados/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) | [Installing Arvados](https://doc.arvados.org/install/index.html) | [Installing Client SDKs](https://doc.arvados.org/sdk/index.html) | [Report a bug](https://dev.arvados.org/projects/arvados/issues/new) | [Development and Contributing](CONTRIBUTING.md)

<img align="right" src="doc/images/dax.png" height="240px">

[Arvados](https://arvados.org) is an open source platform for
managing, processing, and sharing genomic and other large scientific
and biomedical data.  With Arvados, bioinformaticians run and scale
compute-intensive workflows, developers create biomedical
applications, and IT administrators manage large compute and storage
resources.

The key components of Arvados are:

* *Keep*: Keep is the Arvados storage system for managing and storing large
collections of files.  Keep combines content addressing and a
distributed storage architecture resulting in both high reliability
and high throughput.  Every file stored in Keep can be accurately
verified every time it is retrieved.  Keep supports the creation of
collections as a flexible way to define data sets without having to
re-organize or needlessly copy data. Keep works on a wide range of
underlying filesystems and object stores.

* *Crunch*: Crunch is the orchestration system for running [Common Workflow Language](https://www.commonwl.org) workflows. It is
designed to maintain data provenance and workflow
reproducibility. Crunch automatically tracks data inputs and outputs
through Keep and executes workflow processes in Docker containers.  In
a cloud environment, Crunch optimizes costs by scaling compute on demand.

* *Workbench*: The Workbench web application allows users to interactively access
Arvados functionality.  It is especially helpful for querying and
browsing data, visualizing provenance, and tracking the progress of
workflows.

* *Command Line tools*: The command line interface (CLI) provides convenient access to Arvados
functionality in the Arvados platform from the command line.

* *API and SDKs*: Arvados is designed to be integrated with existing infrastructure. All
the services in Arvados are accessed through a RESTful API.  SDKs are
available for Python, Go, R, Perl, Ruby, and Java.

# Documentation

Complete documentation, including the [User Guide](https://doc.arvados.org/user/index.html), [Installation documentation](https://doc.arvados.org/install/index.html), [Administrator documentation](https://doc.arvados.org/admin/index.html) and
[API documentation](https://doc.arvados.org/api/index.html) is available at http://doc.arvados.org/

If you wish to build the Arvados documentation from a local git clone, see
[doc/README.textile](doc/README.textile) for instructions.

# Community

[![Join the chat at https://gitter.im/arvados/community](https://badges.gitter.im/arvados/community.svg)](https://gitter.im/arvados/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

The [Arvados community channel](https://gitter.im/arvados/community)
channel at [gitter.im](https://gitter.im) is available for live
discussion and support.

The [Arvados developement channel](https://gitter.im/arvados/development)
channel at [gitter.im](https://gitter.im) is used to coordinate development.

The [Arvados user mailing list](http://lists.arvados.org/mailman/listinfo/arvados)
is used to announce new versions and other news.

All participants are expected to abide by the [Arvados Code of Conduct](CODE_OF_CONDUCT.md).

# Reporting bugs

[Report a bug](https://dev.arvados.org/projects/arvados/issues/new) on [dev.arvados.org](https://dev.arvados.org).

# Development and Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for information about Arvados development and how to contribute to the Arvados project.

The [development road map](https://dev.arvados.org/issues/gantt?utf8=%E2%9C%93&set_filter=1&gantt=1&f%5B%5D=project_id&op%5Bproject_id%5D=%3D&v%5Bproject_id%5D%5B%5D=49&f%5B%5D=&zoom=1) outlines some of the project priorities over the next twelve months.

# Licensing

Arvados is Free Software.  See [COPYING](COPYING) for information about the open source licenses used in Arvados.
