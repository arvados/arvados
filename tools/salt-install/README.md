[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Arvados install with Saltstack

##### About

This directory holds a small script to install Arvados on a single node, using the
[Saltstack arvados-formula](https://github.com/saltstack-formulas/arvados-formula)
in master-less mode.

The fastest way to get it running is to modify the first lines in the `provision.sh`
script to suit your needs, copy it in the host where you want to install Arvados
and run it as root.

There's an example `Vagrantfile` also, to install it in a vagrant box if you want
to try it locally.

For more information, please read https://doc.arvados.org/v2.1/install/install-using-salt.html
