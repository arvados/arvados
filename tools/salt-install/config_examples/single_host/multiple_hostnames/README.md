Single host with multiple hostnames
===================================

These files let you setup Arvados on a single host using different hostnames
for each of its components nginx's virtualhosts.

The hostnames are composed after the variables "CLUSTER" and "DOMAIN" set in
the `local.params` file.

The virtual hosts' hostnames that will be used are:

* CLUSTER.DOMAIN
* collections.CLUSTER.DOMAIN
* download.CLUSTER.DOMAIN
* keep.CLUSTER.DOMAIN
* keep0.CLUSTER.DOMAIN
* webshell.CLUSTER.DOMAIN
* workbench.CLUSTER.DOMAIN
* workbench2.CLUSTER.DOMAIN
* ws.CLUSTER.DOMAIN
