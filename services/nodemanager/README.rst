====================
Arvados Node Manager
====================

Overview
--------

This package provides ``arvados-node-manager``.  It dynamically starts
and stops compute nodes on an Arvados_ cloud installation based on job
demand.

.. _Arvados: https://arvados.org/

Setup
-----

1. Install the package.

2. Write a configuration file.  ``doc/ec2.example.cfg`` documents all
   of the options available, with specific tunables for EC2 clouds.

3. Run ``arvados-node-manager --config YOURCONFIGFILE`` using whatever
   supervisor you like (e.g., runit).

Testing and Development
-----------------------

To run tests, just run::

  python setup.py test

Our `hacking guide
<https://arvados.org/projects/arvados/wiki/Hacking_Node_Manager>`_
provides an architectural overview of the Arvados Node Manager to help
you find your way around the source.
