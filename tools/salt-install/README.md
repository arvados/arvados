[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Arvados install with Saltstack

##### About

This directory holds a small script to help you get Arvados up and running, using the
[Saltstack arvados-formula](https://git.arvados.org/arvados-formula.git)
in master-less mode.

There are a few preset examples that you can use:

* `single_host`: Install all the Arvados components in a single host. Suitable for testing
  or demo-ing, but not recommended for production use.
* `multi_host/aws`: Let's you install different Arvados components in different hosts on AWS.
  
The fastest way to get it running is to copy the `local.params.example` file to `local.params`,
edit and modify the file to suit your needs, copy this file along with the `provision.sh` script
into the host where you want to install Arvados and run the `provision.sh` script as root.

There's an example `Vagrantfile` also, to install Arvados in a vagrant box if you want
to try it locally.

For more information, please read https://doc.arvados.org/main/install/salt-single-host.html
