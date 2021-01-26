Single host with a single hostname
==================================

These files let you setup Arvados on a single host using a single hostname
for all of its components nginx's virtualhosts.

The hostname MUST be given in the `local.params` file. The script won't try
to guess it because, depending on the network architecture where you're
installing Arvados, things might not work as expected.

The services will be available on the same hostname but different ports,
which can be given on the `local.params` file or will default to the following
values:

* CLUSTER.DOMAIN
* collections
* download
* keep
* keep0
* webshell
* workbench
* workbench2
* ws
