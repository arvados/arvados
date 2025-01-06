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

Installing on Debian and Ubuntu systems
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Arvados publishes packages for Debian 11 "bullseye," Debian 12 "bookworm," Ubuntu 20.04 "focal," and Ubuntu 22.04 "jammy." You can install the Python SDK package on any of these distributions by running the following commands::

  sudo install -d /etc/apt/keyrings
  sudo curl -fsSL -o /etc/apt/keyrings/arvados.asc https://apt.arvados.org/pubkey.gpg
  sudo tee /etc/apt/sources.list.d/arvados.sources >/dev/null <<EOF
  Types: deb
  URIs: https://apt.arvados.org/$(lsb_release -cs)
  Suites: $(lsb_release -cs)
  Components: main
  Signed-by: /etc/apt/keyrings/arvados.asc
  EOF
  sudo apt update
  sudo apt install python3-arvados-fuse

Installing on Red Hat, AlmaLinux, and Rocky Linux
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Arvados publishes packages for RHEL 8 and distributions based on it. Note that these packages depend on, and will automatically enable, the Python 3.9 module. You can install the Python SDK package on any of these distributions by running the following commands::

  sudo tee /etc/yum.repos.d/arvados.repo >/dev/null <<'EOF'
  [arvados]
  name=Arvados
  baseurl=http://rpm.arvados.org/RHEL/$releasever/os/$basearch/
  gpgcheck=1
  gpgkey=http://rpm.arvados.org/RHEL/RPM-GPG-KEY-arvados
  EOF
  sudo dnf install python3-arvados-fuse

Configuration
-------------

This driver needs two pieces of information to connect to
Arvados: the DNS name of the API server, and an API authorization
token.  `The Arvados user
documentation
<http://doc.arvados.org/user/reference/api-tokens.html>`_ describes
how to find this information in the Arvados Workbench, and install it
on your system.

Testing and Development
-----------------------

Debian packages you need to build llfuse::

  sudo apt install python-dev pkg-config libfuse-dev

This package is one part of the Arvados source package, and it has
integration tests to check interoperability with other Arvados
components.  Our `hacking guide
<https://dev.arvados.org/projects/arvados/wiki/Hacking_Python_SDK>`_
describes how to set up a development environment and run tests.
