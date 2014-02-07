#! /bin/bash

build_ok=true

# Check that:
#   * IP forwarding is enabled in the kernel.

if [ "$(/sbin/sysctl --values net.ipv4.ip_forward)" != "1" ]
then
    echo >&2 "WARNING: IP forwarding must be enabled in the kernel."
    echo >&2 "Try: sudo sysctl net.ipv4.ip_forward=1"
    build_ok=false
fi

#   * Docker can be found in the user's path
#   * The user is in the docker group
#   * cgroup is mounted
#   * the docker daemon is running

if ! docker images > /dev/null 2>&1
then
    echo >&2 "WARNING: docker could not be run."
    echo >&2 "Please make sure that:"
    echo >&2 "  * You have permission to read and write /var/run/docker.sock"
    echo >&2 "  * a 'cgroup' volume is mounted on your machine"
    echo >&2 "  * the docker daemon is running"
    build_ok=false
fi

#   * config.yml exists
if [ '!' -f config.yml ]
then
    echo >&2 "WARNING: no config.yml found in the current directory"
    echo >&2 "Copy config.yml.example to config.yml and update it with settings for your site."
    build_ok=false
fi

# If ok to build, then go ahead and run make
if $build_ok
then
    make $*
fi
