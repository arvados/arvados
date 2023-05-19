SSL Certificates
================

Add the certificates for your hosts in this directory.

The nodes requiring certificates are:

* DOMAIN
* collections.DOMAIN
* controller.DOMAIN
* \*.collections.DOMAIN
* grafana.DOMAIN
* download.DOMAIN
* keep.DOMAIN
* prometheus.DOMAIN
* shell.DOMAIN
* workbench.DOMAIN
* workbench2.DOMAIN
* ws.DOMAIN

They can be individual certificates or a wildcard certificate for all of them.

Please remember to modify the *nginx\_\** salt pillars accordingly.
