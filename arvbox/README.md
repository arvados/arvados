# Arvados-in-a-box

Self-contained development, demonstration and testing environment for Arvados.

## Quick start

```
$ bin/arvbox reboot localdemo
```

## Usage

```
Arvados-in-a-box

arvbox (build|start|run|open|shell|ip|stop|reboot|reset|destroy|log|svrestart)

build <config>      build arvbox Docker image
start|run <config>  start arvbox container
open       open arvbox workbench in a web browser
shell      enter arvbox shell
ip         print arvbox ip address
stop       stop arvbox container
restart <config>  stop, then run again
reboot  <config>  stop, build arvbox Docker image, run
reset      delete arvbox arvados data (be careful!)
destroy    delete all arvbox code and data (be careful!)
log       <service> tail log of specified service
svrestart <service> restart specified service inside arvbox
clone <from> <to>   clone an arvbox
```

## Requirements

* Linux 3.x+ and Docker 1.9+
* Minimum of 3 GiB of RAM  + additional memory to run jobs
* Minimum of 3 GiB of disk + storage for actual data

## Configs

### dev
Development configuration.  Boots a complete Arvados environment inside the
container.  The "arvados", "arvado-dev" and "sso-devise-omniauth-provider" code
directories along data directories "postgres", "var", "passenger" and "gems"
are bind mounted from the host file system for easy access and persistence
across container rebuilds.  Services are bound to the Docker container's
network IP address and can only be accessed on the local host.

### localdemo
Demo configuration.  Boots a complete Arvados environment inside the container.
Unlike the development configuration, code directories are included in the demo
image, and data directories are stored in a separate data volume container.
Services are bound to the Docker container's network IP address and can only be
accessed on the local host.

### test
Run the test suite.

### publicdev
Publicly accessible development configuration.  Similar to 'dev' except that
service ports are published to the host's IP address and can accessed by anyone
who can connect to the host system.  WARNING! The public arvbox configuration
is NOT SECURE and must not be placed on a public IP address or used for
production work.

### publicdemo
Publicly accessible development configuration.  Similar to 'localdemo' except
that service ports are published to the host's IP address and can accessed by
anyone who can connect to the host system.  WARNING! The public arvbox configuration
is NOT SECURE and must not be placed on a public IP address or used for
production work.

## Environment variables

### ARVBOX_DOCKER
The location of Dockerfile.base and associated files used by "arvbox build".
default: result of $(readlink -f $(dirname $0)/../lib/arvbox/docker)

### ARVBOX_CONTAINER
The name of the Docker container to manipulate.
default: arvbox

### ARVBOX_BASE
The base directory to store persistent data for arvbox containers.
default: $HOME/.arvbox

### ARVBOX_DATA
The base directory to store persistent data for the current container.
default: $ARVBOX_BASE/$ARVBOX_CONTAINER

### ARVADOS_ROOT
The root directory of the Arvados source tree
default: $ARVBOX_DATA/arvados

### ARVADOS_DEV_ROOT
The root directory of the Arvados-dev source tree
default: $ARVBOX_DATA/arvados-dev

### SSO_ROOT
The root directory of the SSO source tree
default: $ARVBOX_DATA/sso-devise-omniauth-provider

### ARVBOX_PUBLISH_IP
The IP address on which to publish services when running in public
configuration.  Overrides default detection of the host's IP address.

## Notes

Services are designed to install and auto-configure on start or restart.  For
example, the service script for keepstore always compiles keepstore from source
and registers the daemon with the API server.

Services are run with process supervision, so a service which exits will be
restarted.  Dependencies between services are handled by repeatedly trying and
failing the service script until dependencies are fulfilled (by other service
scripts) enabling the service script to complete.
