Deploying Arvados in Docker Containers
======================================

This file explains how to build and deploy Arvados servers in Docker
containers, so that they can be run easily in different environments
(a dedicated server, a developer's laptop, a virtual machine,
etc).

Prerequisites
-------------

* Docker

  Docker is a Linux container management system. It is a very young system but
  is being developed rapidly.
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
	 
  3. [Download and run a docker binary from docker.io.](http://docs.docker.io/en/latest/installation/binaries/)

* Ruby (version 1.9.3 or greater)

* sudo privileges to run `debootstrap`

Building
--------

Type `./build.sh` to configure and build the following Docker images:

   * arvados/api       - the Arvados API server
   * arvados/compute   - Arvados compute node image
   * arvados/doc       - Arvados documentation
   * arvados/keep      - Keep, the Arvados content-addressable filesystem
   * arvados/keepproxy - Keep proxy
   * arvados/shell     - Arvados shell node image
   * arvados/sso       - the Arvados single-signon authentication server
   * arvados/workbench - the Arvados console

`build.sh` will generate reasonable defaults for all configuration
settings.  If you want more control over the way Arvados is
configured, first copy `config.yml.example` to `config.yml` and edit
it with appropriate configuration settings, and then run `./build.sh`.

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
