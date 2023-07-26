.. Copyright (C) The Arvados Authors. All rights reserved.
..
.. SPDX-License-Identifier: AGPL-3.0

========================
Arvados Keep FUSE Driver
========================

Overview
--------

This package provides a FUSE driver for Keep, the Arvados_ storage
system.  It allows you to read data from your collections as if they
were on the local filesystem.

.. _Arvados: https://arvados.org/

Installation
------------

Installing under your user account
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This method lets you install the package without root access.  However,
other users on the same system will need to reconfigure their shell in order
to be able to use it. Run the following to install the package in an
environment at ``~/arvclients``::

  python3 -m venv ~/arvclients
  ~/arvclients/bin/pip install arvados_fuse

Command line tools will be installed under ``~/arvclients/bin``. You can
test one by running::

  ~/arvclients/bin/arv-mount --version

You can run these tools by specifying the full path every time, or you can
add the directory to your shell's search path by running::

  export PATH="$PATH:$HOME/arvclients/bin"

You can make this search path change permanent by adding this command to
your shell's configuration, for example ``~/.bashrc`` if you're using bash.
You can test the change by running::

  arv-mount --version

Installing on Debian systems
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

1. Add this Arvados repository to your sources list::

     deb http://apt.arvados.org/buster buster main

2. Update your package list.

3. Install the ``python3-arvados-fuse`` package.

Configuration
-------------

This driver needs two pieces of information to connect to
Arvados: the DNS name of the API server, and an API authorization
token.  You can set these in environment variables, or the file
``$HOME/.config/arvados/settings.conf``.  `The Arvados user
documentation
<http://doc.arvados.org/user/reference/api-tokens.html>`_ describes
how to find this information in the Arvados Workbench, and install it
on your system.

Testing and Development
-----------------------

Debian packages you need to build llfuse:

$ apt-get install python-dev pkg-config libfuse-dev libattr1-dev

This package is one part of the Arvados source package, and it has
integration tests to check interoperability with other Arvados
components.  Our `hacking guide
<https://dev.arvados.org/projects/arvados/wiki/Hacking_Python_SDK>`_
describes how to set up a development environment and run tests.
