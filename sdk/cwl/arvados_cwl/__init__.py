#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Implement cwl-runner interface for submitting and running work on Arvados, using
# either the Crunch jobs API or Crunch containers API.

import argparse
import logging
import os
import sys
import re
import pkg_resources  # part of setuptools

from schema_salad.sourceline import SourceLine
import schema_salad.validate as validate
import cwltool.main
import cwltool.workflow
import cwltool.process
import cwltool.argparser
from cwltool.process import shortname, UnsupportedRequirement, use_custom_schema
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs, get_listing

import arvados
import arvados.config
from arvados.keep import KeepClient
from arvados.errors import ApiError
import arvados.commands._util as arv_cmd
from arvados.api import OrderedJsonModel

from .perf import Perf
from ._version import __version__
from .executor import ArvCwlExecutor

# These arn't used directly in this file but
# other code expects to import them from here
from .arvcontainer import ArvadosContainer
from .arvjob import ArvadosJob
from .arvtool import ArvadosCommandTool
from .fsaccess import CollectionFsAccess, CollectionCache, CollectionFetcher
from .util import get_current_container
from .executor import RuntimeStatusLoggingHandler, DEFAULT_PRIORITY
from .arvworkflow import ArvadosWorkflow

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')
logger.setLevel(logging.INFO)

arvados.log_handler.setFormatter(logging.Formatter(
        '%(asctime)s %(name)s %(levelname)s: %(message)s',
        '%Y-%m-%d %H:%M:%S'))

def versionstring():
    """Print version string of key packages for provenance and debugging."""

    arvcwlpkg = pkg_resources.require("arvados-cwl-runner")
    arvpkg = pkg_resources.require("arvados-python-client")
    cwlpkg = pkg_resources.require("cwltool")

    return "%s %s, %s %s, %s %s" % (sys.argv[0], arvcwlpkg[0].version,
                                    "arvados-python-client", arvpkg[0].version,
                                    "cwltool", cwlpkg[0].version)


def arg_parser():  # type: () -> argparse.ArgumentParser
    parser = argparse.ArgumentParser(description='Arvados executor for Common Workflow Language')

    parser.add_argument("--basedir", type=str,
                        help="Base directory used to resolve relative references in the input, default to directory of input object file or current directory (if inputs piped/provided on command line).")
    parser.add_argument("--outdir", type=str, default=os.path.abspath('.'),
                        help="Output directory, default current directory")

    parser.add_argument("--eval-timeout",
                        help="Time to wait for a Javascript expression to evaluate before giving an error, default 20s.",
                        type=float,
                        default=20)

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--print-dot", action="store_true",
                         help="Print workflow visualization in graphviz format and exit")
    exgroup.add_argument("--version", action="version", help="Print version and exit", version=versionstring())
    exgroup.add_argument("--validate", action="store_true", help="Validate CWL document only.")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--verbose", action="store_true", help="Default logging")
    exgroup.add_argument("--quiet", action="store_true", help="Only print warnings and errors.")
    exgroup.add_argument("--debug", action="store_true", help="Print even more logging")

    parser.add_argument("--metrics", action="store_true", help="Print timing metrics")

    parser.add_argument("--tool-help", action="store_true", help="Print command line help for tool")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--enable-reuse", action="store_true",
                        default=True, dest="enable_reuse",
                        help="Enable job or container reuse (default)")
    exgroup.add_argument("--disable-reuse", action="store_false",
                        default=True, dest="enable_reuse",
                        help="Disable job or container reuse")

    parser.add_argument("--project-uuid", type=str, metavar="UUID", help="Project that will own the workflow jobs, if not provided, will go to home project.")
    parser.add_argument("--output-name", type=str, help="Name to use for collection that stores the final output.", default=None)
    parser.add_argument("--output-tags", type=str, help="Tags for the final output collection separated by commas, e.g., '--output-tags tag0,tag1,tag2'.", default=None)
    parser.add_argument("--ignore-docker-for-reuse", action="store_true",
                        help="Ignore Docker image version when deciding whether to reuse past jobs.",
                        default=False)

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--submit", action="store_true", help="Submit workflow to run on Arvados.",
                        default=True, dest="submit")
    exgroup.add_argument("--local", action="store_false", help="Run workflow on local host (submits jobs to Arvados).",
                        default=True, dest="submit")
    exgroup.add_argument("--create-template", action="store_true", help="(Deprecated) synonym for --create-workflow.",
                         dest="create_workflow")
    exgroup.add_argument("--create-workflow", action="store_true", help="Create an Arvados workflow (if using the 'containers' API) or pipeline template (if using the 'jobs' API). See --api.")
    exgroup.add_argument("--update-workflow", type=str, metavar="UUID", help="Update an existing Arvados workflow or pipeline template with the given UUID.")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--wait", action="store_true", help="After submitting workflow runner job, wait for completion.",
                        default=True, dest="wait")
    exgroup.add_argument("--no-wait", action="store_false", help="Submit workflow runner job and exit.",
                        default=True, dest="wait")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--log-timestamps", action="store_true", help="Prefix logging lines with timestamp",
                        default=True, dest="log_timestamps")
    exgroup.add_argument("--no-log-timestamps", action="store_false", help="No timestamp on logging lines",
                        default=True, dest="log_timestamps")

    parser.add_argument("--api", type=str,
                        default=None, dest="work_api",
                        choices=("jobs", "containers"),
                        help="Select work submission API.  Default is 'jobs' if that API is available, otherwise 'containers'.")

    parser.add_argument("--compute-checksum", action="store_true", default=False,
                        help="Compute checksum of contents while collecting outputs",
                        dest="compute_checksum")

    parser.add_argument("--submit-runner-ram", type=int,
                        help="RAM (in MiB) required for the workflow runner job (default 1024)",
                        default=None)

    parser.add_argument("--submit-runner-image", type=str,
                        help="Docker image for workflow runner job, default arvados/jobs:%s" % __version__,
                        default=None)

    parser.add_argument("--always-submit-runner", action="store_true",
                        help="When invoked with --submit --wait, always submit a runner to manage the workflow, even when only running a single CommandLineTool",
                        default=False)

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--submit-request-uuid", type=str,
                         default=None,
                         help="Update and commit to supplied container request instead of creating a new one (containers API only).",
                         metavar="UUID")
    exgroup.add_argument("--submit-runner-cluster", type=str,
                         help="Submit workflow runner to a remote cluster (containers API only)",
                         default=None,
                         metavar="CLUSTER_ID")

    parser.add_argument("--collection-cache-size", type=int,
                        default=None,
                        help="Collection cache size (in MiB, default 256).")

    parser.add_argument("--name", type=str,
                        help="Name to use for workflow execution instance.",
                        default=None)

    parser.add_argument("--on-error",
                        help="Desired workflow behavior when a step fails.  One of 'stop' (do not submit any more steps) or "
                        "'continue' (may submit other steps that are not downstream from the error). Default is 'continue'.",
                        default="continue", choices=("stop", "continue"))

    parser.add_argument("--enable-dev", action="store_true",
                        help="Enable loading and running development versions "
                             "of CWL spec.", default=False)
    parser.add_argument('--storage-classes', default="default", type=str,
                        help="Specify comma separated list of storage classes to be used when saving workflow output to Keep.")

    parser.add_argument("--intermediate-output-ttl", type=int, metavar="N",
                        help="If N > 0, intermediate output collections will be trashed N seconds after creation.  Default is 0 (don't trash).",
                        default=0)

    parser.add_argument("--priority", type=int,
                        help="Workflow priority (range 1..1000, higher has precedence over lower, containers api only)",
                        default=DEFAULT_PRIORITY)

    parser.add_argument("--disable-validate", dest="do_validate",
                        action="store_false", default=True,
                        help=argparse.SUPPRESS)

    parser.add_argument("--disable-js-validation",
                        action="store_true", default=False,
                        help=argparse.SUPPRESS)

    parser.add_argument("--thread-count", type=int,
                        default=1, help="Number of threads to use for job submit and output collection.")

    parser.add_argument("--http-timeout", type=int,
                        default=5*60, dest="http_timeout", help="API request timeout in seconds. Default is 300 seconds (5 minutes).")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--trash-intermediate", action="store_true",
                        default=False, dest="trash_intermediate",
                         help="Immediately trash intermediate outputs on workflow success.")
    exgroup.add_argument("--no-trash-intermediate", action="store_false",
                        default=False, dest="trash_intermediate",
                        help="Do not trash intermediate outputs (default).")

    parser.add_argument("workflow", type=str, default=None, help="The workflow to execute")
    parser.add_argument("job_order", nargs=argparse.REMAINDER, help="The input object to the workflow.")

    return parser

def add_arv_hints():
    cwltool.command_line_tool.ACCEPTLIST_EN_RELAXED_RE = re.compile(r".*")
    cwltool.command_line_tool.ACCEPTLIST_RE = cwltool.command_line_tool.ACCEPTLIST_EN_RELAXED_RE
    res = pkg_resources.resource_stream(__name__, 'arv-cwl-schema.yml')
    use_custom_schema("v1.0", "http://arvados.org/cwl", res.read())
    res.close()
    cwltool.process.supportedProcessRequirements.extend([
        "http://arvados.org/cwl#RunInSingleContainer",
        "http://arvados.org/cwl#OutputDirType",
        "http://arvados.org/cwl#RuntimeConstraints",
        "http://arvados.org/cwl#PartitionRequirement",
        "http://arvados.org/cwl#APIRequirement",
        "http://commonwl.org/cwltool#LoadListingRequirement",
        "http://arvados.org/cwl#IntermediateOutput",
        "http://arvados.org/cwl#ReuseRequirement",
        "http://arvados.org/cwl#ClusterTarget"
    ])

def exit_signal_handler(sigcode, frame):
    logger.error("Caught signal {}, exiting.".format(sigcode))
    sys.exit(-sigcode)

def main(args, stdout, stderr, api_client=None, keep_client=None,
         install_sig_handlers=True):
    parser = arg_parser()

    job_order_object = None
    arvargs = parser.parse_args(args)

    if len(arvargs.storage_classes.strip().split(',')) > 1:
        logger.error("Multiple storage classes are not supported currently.")
        return 1

    arvargs.use_container = True
    arvargs.relax_path_checks = True
    arvargs.print_supported_versions = False

    if install_sig_handlers:
        arv_cmd.install_signal_handlers()

    if arvargs.update_workflow:
        if arvargs.update_workflow.find('-7fd4e-') == 5:
            want_api = 'containers'
        elif arvargs.update_workflow.find('-p5p6p-') == 5:
            want_api = 'jobs'
        else:
            want_api = None
        if want_api and arvargs.work_api and want_api != arvargs.work_api:
            logger.error('--update-workflow arg {!r} uses {!r} API, but --api={!r} specified'.format(
                arvargs.update_workflow, want_api, arvargs.work_api))
            return 1
        arvargs.work_api = want_api

    if (arvargs.create_workflow or arvargs.update_workflow) and not arvargs.job_order:
        job_order_object = ({}, "")

    add_arv_hints()

    for key, val in cwltool.argparser.get_default_args().items():
        if not hasattr(arvargs, key):
            setattr(arvargs, key, val)

    try:
        if api_client is None:
            api_client = arvados.safeapi.ThreadSafeApiCache(
                api_params={"model": OrderedJsonModel(), "timeout": arvargs.http_timeout},
                keep_params={"num_retries": 4})
            keep_client = api_client.keep
            # Make an API object now so errors are reported early.
            api_client.users().current().execute()
        if keep_client is None:
            keep_client = arvados.keep.KeepClient(api_client=api_client, num_retries=4)
        executor = ArvCwlExecutor(api_client, arvargs, keep_client=keep_client, num_retries=4)
    except Exception as e:
        logger.error(e)
        return 1

    if arvargs.debug:
        logger.setLevel(logging.DEBUG)
        logging.getLogger('arvados').setLevel(logging.DEBUG)

    if arvargs.quiet:
        logger.setLevel(logging.WARN)
        logging.getLogger('arvados').setLevel(logging.WARN)
        logging.getLogger('arvados.arv-run').setLevel(logging.WARN)

    if arvargs.metrics:
        metrics.setLevel(logging.DEBUG)
        logging.getLogger("cwltool.metrics").setLevel(logging.DEBUG)

    if arvargs.log_timestamps:
        arvados.log_handler.setFormatter(logging.Formatter(
            '%(asctime)s %(name)s %(levelname)s: %(message)s',
            '%Y-%m-%d %H:%M:%S'))
    else:
        arvados.log_handler.setFormatter(logging.Formatter('%(name)s %(levelname)s: %(message)s'))

    return cwltool.main.main(args=arvargs,
                             stdout=stdout,
                             stderr=stderr,
                             executor=executor.arv_executor,
                             versionfunc=versionstring,
                             job_order_object=job_order_object,
                             logger_handler=arvados.log_handler,
                             custom_schema_callback=add_arv_hints,
                             loadingContext=executor.loadingContext,
                             runtimeContext=executor.runtimeContext)
