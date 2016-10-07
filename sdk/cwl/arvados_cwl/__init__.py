#!/usr/bin/env python

# Implement cwl-runner interface for submitting and running work on Arvados, using
# either the Crunch jobs API or Crunch containers API.

import argparse
import logging
import os
import sys
import threading
import hashlib
import copy
import json
from functools import partial
import pkg_resources  # part of setuptools

from cwltool.errors import WorkflowException
import cwltool.main
import cwltool.workflow
import schema_salad

import arvados
import arvados.config

from .arvcontainer import ArvadosContainer, RunnerContainer
from .arvjob import ArvadosJob, RunnerJob, RunnerTemplate
from. runner import Runner
from .arvtool import ArvadosCommandTool
from .arvworkflow import ArvadosWorkflow, upload_workflow
from .fsaccess import CollectionFsAccess
from .perf import Perf
from .pathmapper import FinalOutputPathMapper

from cwltool.pack import pack
from cwltool.process import shortname, UnsupportedRequirement, getListing
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs
from cwltool.draft2tool import compute_checksums
from arvados.api import OrderedJsonModel

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')
logger.setLevel(logging.INFO)


class ArvCwlRunner(object):
    """Execute a CWL tool or workflow, submit work (using either jobs or
    containers API), wait for them to complete, and report output.

    """

    def __init__(self, api_client, work_api=None, keep_client=None, output_name=None):
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
        self.stop_polling = threading.Event()
        self.poll_api = None
        self.pipeline = None
        self.final_output_collection = None
        self.output_name = output_name
        if keep_client is not None:
            self.keep_client = keep_client
        else:
            self.keep_client = arvados.keep.KeepClient(api_client=self.api, num_retries=self.num_retries)

        if self.work_api is None:
            # todo: autodetect API to use.
            self.work_api = "jobs"

        if self.work_api not in ("containers", "jobs"):
            raise Exception("Unsupported API '%s'" % self.work_api)

    def arv_make_tool(self, toolpath_object, **kwargs):
        kwargs["work_api"] = self.work_api
        if "class" in toolpath_object and toolpath_object["class"] == "CommandLineTool":
            return ArvadosCommandTool(self, toolpath_object, **kwargs)
        elif "class" in toolpath_object and toolpath_object["class"] == "Workflow":
            return ArvadosWorkflow(self, toolpath_object, **kwargs)
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
                        with Perf(metrics, "done %s" % j.name):
                            j.done(event["properties"]["new_attributes"])
                        self.cond.notify()
                    finally:
                        self.cond.release()

    def poll_states(self):
        """Poll status of jobs or containers listed in the processes dict.

        Runs in a separate thread.
        """

        while True:
            self.stop_polling.wait(15)
            if self.stop_polling.is_set():
                break
            with self.lock:
                keys = self.processes.keys()
            if not keys:
                continue

            if self.work_api == "containers":
                table = self.poll_api.containers()
            elif self.work_api == "jobs":
                table = self.poll_api.jobs()

            try:
                proc_states = table.list(filters=[["uuid", "in", keys]]).execute(num_retries=self.num_retries)
            except Exception as e:
                logger.warn("Error checking states on API server: %s", e)
                continue

            for p in proc_states["items"]:
                self.on_message({
                    "object_uuid": p["uuid"],
                    "event_type": "update",
                    "properties": {
                        "new_attributes": p
                    }
                })

    def get_uploaded(self):
        return self.uploaded.copy()

    def add_uploaded(self, src, pair):
        self.uploaded[src] = pair

    def check_writable(self, obj):
        if isinstance(obj, dict):
            if obj.get("writable"):
                raise UnsupportedRequirement("InitialWorkDir feature 'writable: true' not supported")
            for v in obj.itervalues():
                self.check_writable(v)
        if isinstance(obj, list):
            for v in obj:
                self.check_writable(v)

    def make_output_collection(self, name, outputObj):
        outputObj = copy.deepcopy(outputObj)

        files = []
        def capture(fileobj):
            files.append(fileobj)

        adjustDirObjs(outputObj, capture)
        adjustFileObjs(outputObj, capture)

        generatemapper = FinalOutputPathMapper(files, "", "", separateDirs=False)

        final = arvados.collection.Collection(api_client=self.api,
                                              keep_client=self.keep_client,
                                              num_retries=self.num_retries)

        srccollections = {}
        for k,v in generatemapper.items():
            sp = k.split("/")
            srccollection = sp[0][5:]
            if srccollection not in srccollections:
                srccollections[srccollection] = arvados.collection.CollectionReader(
                    srccollection,
                    api_client=self.api,
                    keep_client=self.keep_client,
                    num_retries=self.num_retries)
            reader = srccollections[srccollection]
            try:
                srcpath = "/".join(sp[1:]) if len(sp) > 1 else "."
                final.copy(srcpath, v.target, source_collection=reader, overwrite=False)
            except IOError as e:
                logger.warn("While preparing output collection: %s", e)

        def rewrite(fileobj):
            fileobj["location"] = generatemapper.mapper(fileobj["location"]).target
            for k in ("basename", "size", "listing"):
                if k in fileobj:
                    del fileobj[k]

        adjustDirObjs(outputObj, rewrite)
        adjustFileObjs(outputObj, rewrite)

        with final.open("cwl.output.json", "w") as f:
            json.dump(outputObj, f, sort_keys=True, indent=4, separators=(',',': '))

        final.save_new(name=name, owner_uuid=self.project_uuid, ensure_unique_name=True)

        logger.info("Final output collection %s \"%s\" (%s)", final.portable_data_hash(),
                    final.api_response()["name"],
                    final.manifest_locator())

        self.final_output_collection = final

    def arv_executor(self, tool, job_order, **kwargs):
        self.debug = kwargs.get("debug")

        tool.visit(self.check_writable)

        useruuid = self.api.users().current().execute()["uuid"]
        self.project_uuid = kwargs.get("project_uuid") if kwargs.get("project_uuid") else useruuid
        self.pipeline = None
        make_fs_access = kwargs.get("make_fs_access") or partial(CollectionFsAccess,
                                                                 api_client=self.api,
                                                                 keep_client=self.keep_client)
        self.fs_access = make_fs_access(kwargs["basedir"])

        if kwargs.get("create_template"):
            tmpl = RunnerTemplate(self, tool, job_order, kwargs.get("enable_reuse"))
            tmpl.save()
            # cwltool.main will write our return value to stdout.
            return tmpl.uuid

        if kwargs.get("create_workflow") or kwargs.get("update_workflow"):
            return upload_workflow(self, tool, job_order, self.project_uuid, kwargs.get("update_workflow"))

        self.ignore_docker_for_reuse = kwargs.get("ignore_docker_for_reuse")

        kwargs["make_fs_access"] = make_fs_access
        kwargs["enable_reuse"] = kwargs.get("enable_reuse")
        kwargs["use_container"] = True
        kwargs["tmpdir_prefix"] = "tmp"
        kwargs["on_error"] = "continue"
        kwargs["compute_checksum"] = kwargs.get("compute_checksum")

        if self.work_api == "containers":
            kwargs["outdir"] = "/var/spool/cwl"
            kwargs["docker_outdir"] = "/var/spool/cwl"
            kwargs["tmpdir"] = "/tmp"
            kwargs["docker_tmpdir"] = "/tmp"
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
                    runnerjob = RunnerContainer(self, tool, job_order, kwargs.get("enable_reuse"), self.output_name)
            else:
                runnerjob = RunnerJob(self, tool, job_order, kwargs.get("enable_reuse"), self.output_name)

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

        self.poll_api = arvados.api('v1')
        self.polling_thread = threading.Thread(target=self.poll_states)
        self.polling_thread.start()

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

            loopperf = Perf(metrics, "jobiter")
            loopperf.__enter__()
            for runnable in jobiter:
                loopperf.__exit__()
                if runnable:
                    with Perf(metrics, "run"):
                        runnable.run(**kwargs)
                else:
                    if self.processes:
                        self.cond.wait(1)
                    else:
                        logger.error("Workflow is deadlocked, no runnable jobs and not waiting on any pending jobs.")
                        break
                loopperf.__enter__()
            loopperf.__exit__()

            while self.processes:
                self.cond.wait(1)

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
            self.stop_polling.set()
            self.polling_thread.join()

        if self.final_status == "UnsupportedRequirement":
            raise UnsupportedRequirement("Check log for details.")

        if self.final_status != "success":
            raise WorkflowException("Workflow failed.")

        if self.final_output is None:
            raise WorkflowException("Workflow did not return a result.")

        if kwargs.get("submit") and isinstance(runnerjob, Runner):
            logger.info("Final output collection %s", runnerjob.final_output)
        else:
            if self.output_name is None:
                self.output_name = "Output of %s" % (shortname(tool.tool["id"]))
            self.make_output_collection(self.output_name, self.final_output)

        if kwargs.get("compute_checksum"):
            adjustDirObjs(self.final_output, partial(getListing, self.fs_access))
            adjustFileObjs(self.final_output, partial(compute_checksums, self.fs_access))

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

    parser.add_argument("--metrics", action="store_true", help="Print timing metrics")

    parser.add_argument("--tool-help", action="store_true", help="Print command line help for tool")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--enable-reuse", action="store_true",
                        default=True, dest="enable_reuse",
                        help="")
    exgroup.add_argument("--disable-reuse", action="store_false",
                        default=True, dest="enable_reuse",
                        help="")

    parser.add_argument("--project-uuid", type=str, metavar="UUID", help="Project that will own the workflow jobs, if not provided, will go to home project.")
    parser.add_argument("--output-name", type=str, help="Name to use for collection that stores the final output.", default=None)
    parser.add_argument("--ignore-docker-for-reuse", action="store_true",
                        help="Ignore Docker image version when deciding whether to reuse past jobs.",
                        default=False)

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--submit", action="store_true", help="Submit workflow to run on Arvados.",
                        default=True, dest="submit")
    exgroup.add_argument("--local", action="store_false", help="Run workflow on local host (submits jobs to Arvados).",
                        default=True, dest="submit")
    exgroup.add_argument("--create-template", action="store_true", help="Create an Arvados pipeline template.")
    exgroup.add_argument("--create-workflow", action="store_true", help="Create an Arvados workflow.")
    exgroup.add_argument("--update-workflow", type=str, metavar="UUID", help="Update existing Arvados workflow with uuid.")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--wait", action="store_true", help="After submitting workflow runner job, wait for completion.",
                        default=True, dest="wait")
    exgroup.add_argument("--no-wait", action="store_false", help="Submit workflow runner job and exit.",
                        default=True, dest="wait")

    parser.add_argument("--api", type=str,
                        default=None, dest="work_api",
                        help="Select work submission API, one of 'jobs' or 'containers'.")

    parser.add_argument("--compute-checksum", action="store_true", default=False,
                        help="Compute checksum of contents while collecting outputs",
                        dest="compute_checksum")

    parser.add_argument("workflow", type=str, nargs="?", default=None, help="The workflow to execute")
    parser.add_argument("job_order", nargs=argparse.REMAINDER, help="The input object to the workflow.")

    return parser

def add_arv_hints():
    cache = {}
    res = pkg_resources.resource_stream(__name__, 'arv-cwl-schema.yml')
    cache["http://arvados.org/cwl"] = res.read()
    res.close()
    document_loader, cwlnames, _, _ = cwltool.process.get_schema("v1.0")
    _, extnames, _, _ = schema_salad.schema.load_schema("http://arvados.org/cwl", cache=cache)
    for n in extnames.names:
        if not cwlnames.has_name("http://arvados.org/cwl#"+n, ""):
            cwlnames.add_name("http://arvados.org/cwl#"+n, "", extnames.get_name(n, ""))
        document_loader.idx["http://arvados.org/cwl#"+n] = {}

def main(args, stdout, stderr, api_client=None, keep_client=None):
    parser = arg_parser()

    job_order_object = None
    arvargs = parser.parse_args(args)
    if (arvargs.create_template or arvargs.create_workflow or arvargs.update_workflow) and not arvargs.job_order:
        job_order_object = ({}, "")

    add_arv_hints()

    try:
        if api_client is None:
            api_client=arvados.api('v1', model=OrderedJsonModel())
        runner = ArvCwlRunner(api_client, work_api=arvargs.work_api, keep_client=keep_client, output_name=arvargs.output_name)
    except Exception as e:
        logger.error(e)
        return 1

    if arvargs.debug:
        logger.setLevel(logging.DEBUG)

    if arvargs.quiet:
        logger.setLevel(logging.WARN)
        logging.getLogger('arvados.arv-run').setLevel(logging.WARN)

    if arvargs.metrics:
        metrics.setLevel(logging.DEBUG)
        logging.getLogger("cwltool.metrics").setLevel(logging.DEBUG)

    arvargs.conformance_test = None
    arvargs.use_container = True

    return cwltool.main.main(args=arvargs,
                             stdout=stdout,
                             stderr=stderr,
                             executor=runner.arv_executor,
                             makeTool=runner.arv_make_tool,
                             versionfunc=versionstring,
                             job_order_object=job_order_object,
                             make_fs_access=partial(CollectionFsAccess, api_client=api_client))
