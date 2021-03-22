SSL Certificates
================

Add the certificates for your hosts in this directory.

The nodes requiring certificates are:

* CLUSTER.DOMAIN
* collections.CLUSTER.DOMAIN
* \*\-\-collections.CLUSTER.DOMAIN
* download.CLUSTER.DOMAIN
* keep.CLUSTER.DOMAIN
* workbench.CLUSTER.DOMAIN
* workbench2.CLUSTER.DOMAIN
* ws.CLUSTER.DOMAIN

They can be individual certificates or a wildcard certificate for all of them.

Please remember to modify the *nginx\_\** salt pillars accordingly.
