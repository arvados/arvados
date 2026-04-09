# Arvados Client Tools

<!-- Copyright (C) The Arvados Authors. All rights reserved.

SPDX-License-Identifier: Apache-2.0 -->

## Overview

This is a metapackage that lets you install all of the Python client tools for [Arvados][] in one simple command. It's intended for users setting up an interactive environment. It provides:

* the [Arvados Python SDK](https://doc.arvados.org/sdk/python/api-client.html)
* command line tools to work with collections and projects: [`arv-ls`, `arv-get`](https://doc.arvados.org/user/tutorials/tutorial-keep-get.html#download-using-arv), [`arv-put`](https://doc.arvados.org/user/tutorials/tutorial-keep.html#upload-using-command), [`arv-copy`](https://doc.arvados.org/user/topics/arv-copy.html), and [`arv-mount`](https://doc.arvados.org/user/tutorials/tutorial-keep-mount-gnu-linux.html)
* [workflow runners `arvados-cwl-runner`](https://doc.arvados.org/user/cwl/cwl-runner.html) and [`cwltool`](https://pypi.org/project/cwltool/)
* reporting tools for [workflow performance](https://doc.arvados.org/user/cwl/crunchstat-summary.html), [cluster activity](https://doc.arvados.org/user/cwl/costanalyzer.html), and [user activity](https://doc.arvados.org/admin/user-activity.html)

If you are building your own Arvados client software, it is better to require the specific package(s) you need like [arvados-python-client](https://pypi.org/project/arvados-python-client/).

[Arvados]: https://arvados.org/

## Installation

We recommend you install with `pipx`. First [install `pipx`][install-pipx] (it's available as the `pipx` package in most Linux distributions). Then run:

      pipx install --include-deps arvados-tools

[install-pipx]: https://pipx.pypa.io/latest/how-to/install-pipx/

Alternatively, if you're comfortable setting up your own virtual environments, you can install the package in one too. For example:

      python3 -m venv MYVENV
      MYVENV/bin/pip install arvados-tools

Now all the Arvados tools will be available after you activate `MYVENV` with `source MYVENV/bin/activate`.

## Configuration

This client software needs two pieces of information to connect to Arvados: the DNS name of the API server, and an API authorization token. [The Arvados user documentation](http://doc.arvados.org/user/reference/api-tokens.html) describes how to find this information in the Arvados Workbench, and install it on your system.

## Licenses

The SDK and most command line tools installed are published under the [Apache License 2.0](https://spdx.org/licenses/Apache-2.0.html). `arv-mount` and the reporting tools are published under the [GNU Affero General Public License 3.0](https://spdx.org/licenses/AGPL-3.0-only.html). Refer to the individual component packages for details.
