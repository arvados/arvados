#!/usr/bin/env python

# Implement cwl-runner interface for submitting and running work on Arvados, using
# either the Crunch jobs API or Crunch containers API.

import argparse
import logging
import os
import sys
import threading
import pkg_resources  # part of setuptools

from cwltool.errors import WorkflowException
import cwltool.main
import cwltool.workflow

import arvados
import arvados.events
import arvados.config

from .arvcontainer import ArvadosContainer, RunnerContainer
from .arvjob import ArvadosJob, RunnerJob, RunnerTemplate
from .arvtool import ArvadosCommandTool
from .fsaccess import CollectionFsAccess

from cwltool.process import shortname, UnsupportedRequirement
from arvados.api import OrderedJsonModel

logger = logging.getLogger('arvados.cwl-runner')
logger.setLevel(logging.INFO)

class ArvCwlRunner(object):
    """Execute a CWL tool or workflow, submit work (using either jobs or
    containers API), wait for them to complete, and report output.

    """

    def __init__(self, api_client, work_api=None):
        self.api = api_client
        self.processes = {}
        self.lock = threading.Lock()
        self.cond = threading.Condition(self.lock)
        self.final_output = None
        self.final_status = None
        self.uploaded = {}
        self.num_retries = 4
        self.uuid = None
        self.work_api = work_api

        if self.work_api is None:
            # todo: autodetect API to use.
            self.work_api = "jobs"

        if self.work_api not in ("containers", "jobs"):
            raise Exception("Unsupported API '%s'" % self.work_api)

    def arvMakeTool(self, toolpath_object, **kwargs):
        if "class" in toolpath_object and toolpath_object["class"] == "CommandLineTool":
            return ArvadosCommandTool(self, toolpath_object, work_api=self.work_api, **kwargs)
        else:
            return cwltool.workflow.defaultMakeTool(toolpath_object, **kwargs)

    def output_callback(self, out, processStatus):
        if processStatus == "success":
            logger.info("Overall process status is %s", processStatus)
            if self.pipeline:
                self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                     body={"state": "Complete"}).execute(num_retries=self.num_retries)
        else:
            logger.warn("Overall process status is %s", processStatus)
            if self.pipeline:
                self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                     body={"state": "Failed"}).execute(num_retries=self.num_retries)
        self.final_status = processStatus
        self.final_output = out

    def on_message(self, event):
        if "object_uuid" in event:
            if event["object_uuid"] in self.processes and event["event_type"] == "update":
                if event["properties"]["new_attributes"]["state"] == "Running" and self.processes[event["object_uuid"]].running is False:
                    uuid = event["object_uuid"]
                    with self.lock:
                        j = self.processes[uuid]
                        logger.info("Job %s (%s) is Running", j.name, uuid)
                        j.running = True
                        j.update_pipeline_component(event["properties"]["new_attributes"])
                elif event["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled"):
                    uuid = event["object_uuid"]
                    try:
                        self.cond.acquire()
                        j = self.processes[uuid]
                        logger.info("Job %s (%s) is %s", j.name, uuid, event["properties"]["new_attributes"]["state"])
                        j.done(event["properties"]["new_attributes"])
                        self.cond.notify()
                    finally:
                        self.cond.release()

    def get_uploaded(self):
        return self.uploaded.copy()

    def add_uploaded(self, src, pair):
        self.uploaded[src] = pair

    def arvExecutor(self, tool, job_order, **kwargs):
        self.debug = kwargs.get("debug")

        if kwargs.get("quiet"):
            logger.setLevel(logging.WARN)
            logging.getLogger('arvados.arv-run').setLevel(logging.WARN)

        useruuid = self.api.users().current().execute()["uuid"]
        self.project_uuid = kwargs.get("project_uuid") if kwargs.get("project_uuid") else useruuid
        self.pipeline = None
        self.fs_access = CollectionFsAccess(kwargs["basedir"], api_client=self.api)

        if kwargs.get("create_template"):
            tmpl = RunnerTemplate(self, tool, job_order, kwargs.get("enable_reuse"))
            tmpl.save()
            # cwltool.main will write our return value to stdout.
            return tmpl.uuid

        self.debug = kwargs.get("debug")
        self.ignore_docker_for_reuse = kwargs.get("ignore_docker_for_reuse")

        kwargs["fs_access"] = self.fs_access
        kwargs["enable_reuse"] = kwargs.get("enable_reuse")
        kwargs["use_container"] = True
        kwargs["tmpdir_prefix"] = "tmp"
        kwargs["on_error"] = "continue"

        if self.work_api == "containers":
            kwargs["outdir"] = "/var/spool/cwl"
            kwargs["docker_outdir"] = "/var/spool/cwl"
            kwargs["tmpdir"] = "/tmp"
        elif self.work_api == "jobs":
            kwargs["outdir"] = "$(task.outdir)"
            kwargs["docker_outdir"] = "$(task.outdir)"
            kwargs["tmpdir"] = "$(task.tmpdir)"

        runnerjob = None
        if kwargs.get("submit"):
            if self.work_api == "containers":
                if tool.tool["class"] == "CommandLineTool":
                    runnerjob = tool.job(job_order,
                                         self.output_callback,
                                         **kwargs).next()
                else:
                    runnerjob = RunnerContainer(self, tool, job_order, kwargs.get("enable_reuse"))
            else:
                runnerjob = RunnerJob(self, tool, job_order, kwargs.get("enable_reuse"))

        if not kwargs.get("submit") and "cwl_runner_job" not in kwargs and not self.work_api == "containers":
            # Create pipeline for local run
            self.pipeline = self.api.pipeline_instances().create(
                body={
                    "owner_uuid": self.project_uuid,
                    "name": shortname(tool.tool["id"]),
                    "components": {},
                    "state": "RunningOnClient"}).execute(num_retries=self.num_retries)
            logger.info("Pipeline instance %s", self.pipeline["uuid"])

        if runnerjob and not kwargs.get("wait"):
            runnerjob.run()
            return runnerjob.uuid

        arvados.config.settings()["ARVADOS_DISABLE_WEBSOCKETS"] = "1"

        if self.work_api == "containers":
            events = arvados.events.subscribe(arvados.api('v1'), [["object_uuid", "is_a", "arvados#container"]], self.on_message)
        if self.work_api == "jobs":
            events = arvados.events.subscribe(arvados.api('v1'), [["object_uuid", "is_a", "arvados#job"]], self.on_message)

        if runnerjob:
            jobiter = iter((runnerjob,))
        else:
            if "cwl_runner_job" in kwargs:
                self.uuid = kwargs.get("cwl_runner_job").get('uuid')
            jobiter = tool.job(job_order,
                               self.output_callback,
                               **kwargs)

        try:
            self.cond.acquire()
            # Will continue to hold the lock for the duration of this code
            # except when in cond.wait(), at which point on_message can update
            # job state and process output callbacks.

            for runnable in jobiter:
                if runnable:
                    runnable.run(**kwargs)
                else:
                    if self.processes:
                        self.cond.wait(1)
                    else:
                        logger.error("Workflow is deadlocked, no runnable jobs and not waiting on any pending jobs.")
                        break

            while self.processes:
                self.cond.wait(1)

            events.close()
        except UnsupportedRequirement:
            raise
        except:
            if sys.exc_info()[0] is KeyboardInterrupt:
                logger.error("Interrupted, marking pipeline as failed")
            else:
                logger.error("Caught unhandled exception, marking pipeline as failed.  Error was: %s", sys.exc_info()[1], exc_info=(sys.exc_info()[1] if self.debug else False))
            if self.pipeline:
                self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                     body={"state": "Failed"}).execute(num_retries=self.num_retries)
            if runnerjob and runnerjob.uuid and self.work_api == "containers":
                self.api.container_requests().update(uuid=runnerjob.uuid,
                                                     body={"priority": "0"}).execute(num_retries=self.num_retries)
        finally:
            self.cond.release()

        if self.final_status == "UnsupportedRequirement":
            raise UnsupportedRequirement("Check log for details.")

        if self.final_status != "success":
            raise WorkflowException("Workflow failed.")

        if self.final_output is None:
            raise WorkflowException("Workflow did not return a result.")

        return self.final_output


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
    parser.add_argument("--version", action="store_true", help="Print version and exit")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--verbose", action="store_true", help="Default logging")
    exgroup.add_argument("--quiet", action="store_true", help="Only print warnings and errors.")
    exgroup.add_argument("--debug", action="store_true", help="Print even more logging")

    parser.add_argument("--tool-help", action="store_true", help="Print command line help for tool")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--enable-reuse", action="store_true",
                        default=True, dest="enable_reuse",
                        help="")
    exgroup.add_argument("--disable-reuse", action="store_false",
                        default=True, dest="enable_reuse",
                        help="")

    parser.add_argument("--project-uuid", type=str, help="Project that will own the workflow jobs, if not provided, will go to home project.")
    parser.add_argument("--ignore-docker-for-reuse", action="store_true",
                        help="Ignore Docker image version when deciding whether to reuse past jobs.",
                        default=False)

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--submit", action="store_true", help="Submit workflow to run on Arvados.",
                        default=True, dest="submit")
    exgroup.add_argument("--local", action="store_false", help="Run workflow on local host (submits jobs to Arvados).",
                        default=True, dest="submit")
    exgroup.add_argument("--create-template", action="store_true", help="Create an Arvados pipeline template.")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--wait", action="store_true", help="After submitting workflow runner job, wait for completion.",
                        default=True, dest="wait")
    exgroup.add_argument("--no-wait", action="store_false", help="Submit workflow runner job and exit.",
                        default=True, dest="wait")

    parser.add_argument("--api", type=str,
                        default=None, dest="work_api",
                        help="Select work submission API, one of 'jobs' or 'containers'.")

    parser.add_argument("workflow", type=str, nargs="?", default=None, help="The workflow to execute")
    parser.add_argument("job_order", nargs=argparse.REMAINDER, help="The input object to the workflow.")

    return parser


def main(args, stdout, stderr, api_client=None):
    parser = arg_parser()

    job_order_object = None
    arvargs = parser.parse_args(args)
    if arvargs.create_template and not arvargs.job_order:
        job_order_object = ({}, "")

    try:
        if api_client is None:
            api_client=arvados.api('v1', model=OrderedJsonModel())
        runner = ArvCwlRunner(api_client, work_api=arvargs.work_api)
    except Exception as e:
        logger.error(e)
        return 1

    arvargs.conformance_test = None
    arvargs.use_container = True

    return cwltool.main.main(args=arvargs,
                             stdout=stdout,
                             stderr=stderr,
                             executor=runner.arvExecutor,
                             makeTool=runner.arvMakeTool,
                             versionfunc=versionstring,
                             job_order_object=job_order_object)
