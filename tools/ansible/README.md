# Arvados Ansible Playbooks

<!--
Copyright (C) The Arvados Authors. All rights reserved.

SPDX-License-Identifier: Apache-2.0
-->

This directory includes Ansible playbooks and supporting infrastructure to automate various aspects of Arvados deployment.

## Installing Ansible

### Install with pipx

Installing with pipx is the recommended method: it automatically manages a virtualenv for you and adds installed tools to your `$PATH`. Install `pipx` from your distribution, then run:

      ./install-ansible.sh

### Install to your own virtualenv

If you need to keep this Ansible install isolated, you can install it to a virtualenv you set up. You'll need to activate this virtualenv when you want to run Arvados Ansible playbooks.

Make sure you have Python and its standard `venv` module installed from your distribution. You should be able to run `python3 -m venv --help`. (On Debian/Ubuntu, `apt install python3-venv`.) Then run:

      ./install-ansible.sh VENV_DIR

`VENV_DIR` can be any path you like. If you already have a virtualenv activated, you can install inside it by running:

      ./install-ansible.sh -V

### Manual installation

If you need to orchestrate your own install, you must install the Python packages listed in `requirements.txt`, then the Ansible collections listed in `requirements.yml`.
