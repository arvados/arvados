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

This method lets you install the package without root access.
However, other users on the same system won't be able to use it.

1. Run ``pip install --user arvados_fuse``.

2. In your shell configuration, make sure you add ``$HOME/.local/bin``
   to your PATH environment variable.  For example, you could add the
   command ``PATH=$PATH:$HOME/.local/bin`` to your ``.bashrc`` file.

3. Reload your shell configuration.  For example, bash users could run
   ``source ~/.bashrc``.

Installing on Debian systems
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

1. Add this Arvados repository to your sources list::

     deb http://apt.arvados.org/ wheezy main

2. Update your package list.

3. Install the ``python-arvados-fuse`` package.

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

This package is one part of the Arvados source package, and it has
integration tests to check interoperability with other Arvados
components.  Our `hacking guide
<https://arvados.org/projects/arvados/wiki/Hacking_Python_SDK>`_
describes how to set up a development environment and run tests.
