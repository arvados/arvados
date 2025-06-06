# Arvados Ansible Playbooks

<!--
Copyright (C) The Arvados Authors. All rights reserved.

SPDX-License-Identifier: Apache-2.0
-->

This directory includes Ansible playbooks and supporting infrastructure to automate various aspects of Arvados deployment.

## Installation

These instructions set up a virtualenv at `~/ansible`, but you can use any path you like.

1. Create a virtualenv:

        $ python3 -m venv ~/ansible

2. Activate the virtualenv:

        $ . ~/ansible/bin/activate

3. Install required Python modules:

        (ansible) arvados/tools/ansible $ pip install -r requirements.txt

4. Install required Ansible collections:

        (ansible) arvados/tools/ansible $ ansible-galaxy install -r requirements.yml

Now you can run `ansible-playbook` and other tools from `~/ansible/bin`.
