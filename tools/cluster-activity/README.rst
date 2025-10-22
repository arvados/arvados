.. Copyright (C) The Arvados Authors. All rights reserved.
..
.. SPDX-License-Identifier: AGPL-3.0

=================================
 Arvados Cluster Activity Report
=================================

This tool reports on the data and workflows in an Arvados cluster to help administrators understand growth and costs. It reports what it has access to: any Arvados user can run it to get a report of their own workflows and others they can see. An Arvados administrator can run a report on all data and workflows in the cluster. If you provide credentials for a Prometheus server in your Arvados cluster, the report includes additional information about compute use.

Running as a workflow from Workbench
====================================

We provide a CWL workflow to generate this report. It's available as a `single file in the Arvados source`_ and included with this Python package. You can register the workflow on your cluster by running::

  arvados-cwl-runner [--project-uuid=UUID] --create-workflow cluster-activity.cwl

Then you can launch the workflow from Workbench. All inputs have documented formats and values.

Running as a workflow from the command line
===========================================

Alternatively, you can run the workflow directly with ``arvados-cwl-runner``. Write an input file following this YAML template::

  # Report start date as a `YYYY-MM-DD` string
  reporting_start: "YYYY-MM-DD"

  # Report end date as a `YYYY-MM-DD` string. Default today.
  #reporting_end: "YYYY-MM-DD"

  # The base URL of your Arvados cluster's Prometheus server, like
  # `https://prometheus.arvados.example/`
  #prometheus_host: ""

  # Prometheus API token
  #prometheus_apikey: ""

  # Prometheus API username
  #prometheus_user: ""

  # Prometheus API password
  #prometheus_password: ""

  # A string with a Python regular expression.
  # Workflows whose name match the expression will be excluded from the report.
  #exclude: ""

  # A boolean. If true, individual workflow steps will be reported alongside
  # their parent workflows.
  include_workflow_steps: false

Then run `the workflow`_ like this::

  arvados-cwl-runner [--project-uuid=UUID] [options ...] cluster-activity.cwl YOUR-INPUTS.yml

.. _the workflow: https://github.com/arvados/arvados/blob/main/tools/cluster-activity/cluster-activity.cwl
.. _single file in the Arvados source: `the workflow`_

Running as a command line tool
==============================

This Python package provides a command line tool you can run to generate reports on your own system. Install it with `pipx`_ like::

  pipx install "arvados-cluster-activity[prometheus]"

If you don't have a Prometheus server or don't want Prometheus support, remove ``[prometheus]`` from the command line. Advanced users can install the tool to their own virtualenv or elsewhere.

The command line tool provides options to control the report generation. These correspond to the workflow inputs. Run the tool with ``--help`` for the full list::

  arv-cluster-activity --help

The tool gets Arvados credentials the same as other client tools: it reads the ``ARVADOS_API_HOST`` and ``ARVADOS_API_TOKEN`` environment variables if those are set, or the ``~/.config/arvados/settings.conf`` file if they are not.

The tool gets Prometheus credentials from the ``PROMETHEUS_HOST``, ``PROMETHEUS_APIKEY``, ``PROMETHEUS_USER``, and ``PROMETHEUS_PASSWORD`` environment variables. The values follow the format of the workflow inputs above. You can write these environment variables in a dedicated file and load that with the tool's ``--prometheus-auth`` option.

.. _pipx: https://pipx.pypa.io/stable/
