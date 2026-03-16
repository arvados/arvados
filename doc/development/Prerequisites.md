[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Hacking prerequisites

This page describes how to install all the software necessary to develop Arvados and run tests.

## Host options

You must have a system running a supported distribution. That system can be installed directly on hardware; running on a cloud instance; or in a virtual machine.

### Supported distributions

As of March 2026/Arvados 3.2, these instructions and the entire test suite are known to work on Debian 12 "bookworm" and Debian 13 “trixie.”

You may try to run these instructions and tests on Ubuntu 22.04 “jammy”/24.04 “noble,” but they have not been tested and you may find some bugs throughout.

These instructions are not suitable for any Red Hat-based distribution. Our Ansible playbook will refuse to run on them.

### Base configuration

On your development system, you should have a user account with full permission to use sudo.

You can run the Ansible playbook to install your development system on a different system. To do this, you must have permission to SSH into your user account from the system running Ansible (the “control node”) to the development system you’re installing (the “target node”).

### Virtual machine requirements

If you run your development system in a virtual machine, it needs some permissions. Many environments will allow these operations by default, but they could be limited by your virtual machine setup.

- It must be able to create and manage FUSE mounts (`/dev/fuse`)
- It must be able to create and run Docker containers
- It must be able to create and run Singularity containers—this requires creating and managing block loopback devices (`/dev/block-loop`)
- It must have the `fs.inotify.max_user_watches` sysctl set to at least 524288. Our Ansible playbook will try to set this on the managed host, but if it is unable to do so, you may need to set it on the parent host instead.

## Install development environment with Ansible

### Clone Arvados source

You will need the Arvados source code to follow this process.

```sh
$ git clone https://github.com/arvados/arvados.git
```

If you want to switch to a specific branch or revision like `3.2-release`, do that here.

### Install Ansible

Install Ansible following the instructions in `arvados/tools/ansible/README.md`. This ensures you get the right versions of everything.

### Write an Arvados database configuration

Make a copy of the default test configuration:

```sh
$ cp arvados/tools/ansible/files/default-test-config.yml ~/zzzzz-config.yml
```

You can copy the file to a different location if you like. This page will use `~/zzzzz-config.yml` as the placeholder path throughout.

Edit this file with the database configuration you’d like to use. The cluster ID **must** be `zzzzz`. You can change the `user`, `password`, and `dbname` settings freely. Our Ansible playbook will configure PostgreSQL so your settings here work.

The playbook will always install the `postgresql` server package. It will **not** change any PostgreSQL configuration except to add `pg_hba.conf` entries for this user. You should only change `host` and `port` if you need to use a PostgreSQL server that is already installed and running somewhere else.

### Write an Ansible inventory

An inventory file tells Ansible what host(s) to manage, how to connect to them, and what settings they use. Write an inventory file to `~/zzzzz-inventory.yml` like this:

```yaml
arvados_test_all:
  # This is the list of host(s) where we're installing the test environment.
  # This example installs on the same system running Ansible.
  # If you want to manage remote hosts, you can write your own host list:
  # <https://docs.ansible.com/ansible/latest/getting_started/get_started_inventory.html>
  hosts:
    localhost:
      ansible_connection: local
  vars:
    # The path to the Arvados cluster configuration you wrote in the previous section.
    arvados_config_file: "{{ lookup('env', 'HOME') }}/zzzzz-config.yml"

    # The primary user doing Arvados development and tests.
    # This user will be added to the `docker` group.
    # It defaults to the name of the user running `ansible-playbook`.
    # If you want to configure a different user, set that here:
    #arvados_dev_user: USERNAME

    # By default, the playbook installs old versions of Python and Ruby from source.
    # This helps you make sure you don't accidentally use too-new features during
    # development. If you're sure you don't need that—for example, you specifically
    # want to test a distribution's packaged version—set this flag:
    #arvados_dev_from_pkgs: true
```

### Run the playbook

The basic command to run the playbook is:

```sh
$ cd arvados/tools/ansible
$ ansible-playbook -K -i ~/zzzzz-inventory.yml install-dev-tools.yml
```

When you are prompted for the `BECOME password:`, enter the password for your user account on the development host that lets you run `sudo` commands.

`ansible-playbook` has many options to control how it runs that you can add if you like. Refer to [the `ansible-playbook` documentation](https://docs.ansible.com/ansible/latest/cli/ansible-playbook.html) for more information.

## Run Arvados tests

After the playbook runs successfully, you should be able to run the Arvados tests from a source checkout on your development host. This document will walk you through setting up and running a single test suite to verify your setup. `cd` to your Arvados checkout and run:

```sh
$ mkdir -p tmp/run-tests
$ build/run-tests.sh --temp "$PWD/tmp/run-tests" --interactive
```

This will install baseline prerequisites, then list commands and test targets, then prompt you with:

    What next? install deps

Accept that command. It will install the rest of the dependencies that are necessary for running a test cluster, then report:

    All test suites passed.

At this stage, this message simply means that the "install deps" command has succeeded.

Now we can run a test suite. The controller tests are a good first example, because they interact with a test cluster but not much else. At the `What next?` prompt, enter `test lib/controller`, and you'll see the test cluster start:

    What next? test lib/controller
    Starting API, controller, keepproxy, keep-web, ws, and nginx ssl proxy...

You'll see logs from individual services, then, hopefully, the controller tests starting and passing:

    ======= test lib/controller
    ok  	git.arvados.org/arvados.git/lib/controller	64.679s	coverage: 82.0% of statements
    ======= test lib/controller -- 68s
    Pass: lib/controller tests (68s)
    All test suites passed.

Refer to [Running tests](RunningTests.md) for details about running specific test suites, test selection, and other features.

## Troubleshooting

If the playbook succeeds but you can't get tests running, there might be a disconnect between your shell configuration and what the system expects. This section documents some places you can look.

### Dependencies in `$PATH`

The playbook will install symlinks for Go, Node, Python, Ruby, Singularity, and Yarn under `/usr/local/bin`. The actual tools are installed under `/opt`. When you run Arvados tests or other development tools, you must ensure `/usr/local/bin` appears in your `$PATH` before any directories with other versions like `/usr/bin`.

### Arvados `$CONFIGSRC`

The playbook writes the Arvados test cluster's database configuration at `~/.config/arvados/config.yml`, and sets up a hook `/etc/profile.d/arvados-test.sh` to set your `CONFIGSRC` environment variable to that file's base directory. If most tests fail with a database connection error, check that this variable is set:

```sh
$ echo "${CONFIGSRC:-UNSET}"
/home/you/.config/arvados
```

If that reports `UNSET`, first check if you're using a stale shell session started before the Ansible playbook run. You may need to log out of that session and start a new one.

If that doesn't work, you may add a line to set `CONFIGSRC="$HOME/.config/arvados"` to your shell configuration, or set it manually when you run `run-tests.sh`:

```sh
$ CONFIGSRC="$HOME/.config/arvados" build/run-tests.sh ...
```
