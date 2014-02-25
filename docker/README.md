Deploying Arvados in Docker Containers
======================================

This file explains how to build and deploy Arvados servers in Docker
containers, so that they can be run easily in different environments
(a dedicated server, a developer's laptop, a virtual machine,
etc).

This is a work in progress; instructions will almost certainly be
incomplete and possibly out of date.

Prerequisites
-------------

* Docker

  Docker is a Linux container management system based on LXC. It is a
  very young system but is being developed rapidly.
  [Installation packages](http://www.docker.io/gettingstarted/)
  are available for several platforms.
  
  If a prebuilt docker package is not available for your platform, the
  short instructions for installing it are:
  
  1. Create a `docker` group and add yourself to it.

     <pre>
     $ sudo addgroup docker
     $ sudo adduser `whoami` docker
     </pre>

     Log out and back in.
	 
  2. Add a `cgroup` filesystem and mount it:

     <pre>
     $ mkdir -p /cgroup
     $ grep cgroup /etc/fstab
     none   /cgroup    cgroup    defaults    0    0
     $ sudo mount /cgroup
	 </pre>

  3. Enable IPv4 forwarding:

     <pre>
     $ grep ipv4.ip_forward /etc/sysctl.conf
     net.ipv4.ip_forward=1
     $ sudo sysctl net.ipv4.ip_forward=1
     </pre>
	 
  4. [Download and run a docker binary from docker.io.](http://docs.docker.io/en/latest/installation/binaries/)

* Ruby (any version)

* sudo privileges to run `debootstrap`

Building
--------

1. Copy `config.yml.example` to `config.yml` and edit it with settings
   for your installation.
2. Run `make` to build the following Docker images:

   * arvados/api       - the Arvados API server
   * arvados/doc       - Arvados documentation
   * arvados/warehouse - Keep, the Arvados content-addressable filesystem
   * arvados/workbench - the Arvados console
   * arvados/sso       - the Arvados single-signon authentication server

   You may also build Docker images for individual Arvados services:

        $ make api-image
        $ make doc-image
        $ make warehouse-image
        $ make workbench-image
        $ make sso-image

Running
-------

The `arvdock` script in this directory is used to start, stop and
restart Arvados servers on your machine.  The simplest and easiest way
to use it is `./arvdock start` to start the full complement of Arvados
servers, and `./arvdock stop` and `./arvdock restart` to stop and
restart all servers, respectively.

Developers who are working on individual servers can start, stop or
restart just those containers, e.g.:

* `./arvdock start --api --sso` to start just the API and SSO services.
* `./arvdock stop --keep` to stop just the Keep services.
* `./arvdock restart --workbench=8000` restarts just the Workbench service on port 8000.

For a full set of arguments, use `./arvdock --help`.
