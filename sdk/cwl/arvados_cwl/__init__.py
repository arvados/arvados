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
import re
from functools import partial
import pkg_resources  # part of setuptools

from cwltool.errors import WorkflowException
import cwltool.main
import cwltool.workflow
import cwltool.process
import schema_salad
from schema_salad.sourceline import SourceLine

import arvados
import arvados.config
from arvados.keep import KeepClient
from arvados.errors import ApiError

from .arvcontainer import ArvadosContainer, RunnerContainer
from .arvjob import ArvadosJob, RunnerJob, RunnerTemplate
from. runner import Runner, upload_docker, upload_job_order, upload_workflow_deps, upload_dependencies
from .arvtool import ArvadosCommandTool
from .arvworkflow import ArvadosWorkflow, upload_workflow
from .fsaccess import CollectionFsAccess, CollectionFetcher, collectionResolver, CollectionCache
from .perf import Perf
from .pathmapper import NoFollowPathMapper
from ._version import __version__

from cwltool.pack import pack
from cwltool.process import shortname, UnsupportedRequirement, use_custom_schema
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs, get_listing
from cwltool.draft2tool import compute_checksums
from arvados.api import OrderedJsonModel

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')
logger.setLevel(logging.INFO)

arvados.log_handler.setFormatter(logging.Formatter(
        '%(asctime)s %(name)s %(levelname)s: %(message)s',
        '%Y-%m-%d %H:%M:%S'))

class ArvCwlRunner(object):
    """Execute a CWL tool or workflow, submit work (using either jobs or
    containers API), wait for them to complete, and report output.

    """

    def __init__(self, api_client, work_api=None, keep_client=None, output_name=None, output_tags=None, num_retries=4):
        self.api = api_client
        self.processes = {}
        self.lock = threading.Lock()
        self.cond = threading.Condition(self.lock)
        self.final_output = None
        self.final_status = None
        self.uploaded = {}
        self.num_retries = num_retries
        self.uuid = None
        self.stop_polling = threading.Event()
        self.poll_api = None
        self.pipeline = None
        self.final_output_collection = None
        self.output_name = output_name
        self.output_tags = output_tags
        self.project_uuid = None
        self.intermediate_output_ttl = 0
        self.intermediate_output_collections = []
        self.trash_intermediate = False

        if keep_client is not None:
            self.keep_client = keep_client
        else:
            self.keep_client = arvados.keep.KeepClient(api_client=self.api, num_retries=self.num_retries)

        self.collection_cache = CollectionCache(self.api, self.keep_client, self.num_retries)

        self.work_api = None
        expected_api = ["jobs", "containers"]
        for api in expected_api:
            try:
                methods = self.api._rootDesc.get('resources')[api]['methods']
                if ('httpMethod' in methods['create'] and
                    (work_api == api or work_api is None)):
                    self.work_api = api
                    break
            except KeyError:
                pass

        if not self.work_api:
            if work_api is None:
                raise Exception("No supported APIs")
            else:
                raise Exception("Unsupported API '%s', expected one of %s" % (work_api, expected_api))

    def arv_make_tool(self, toolpath_object, **kwargs):
        kwargs["work_api"] = self.work_api
        kwargs["fetcher_constructor"] = partial(CollectionFetcher,
                                                api_client=self.api,
                                                fs_access=CollectionFsAccess("", collection_cache=self.collection_cache),
                                                num_retries=self.num_retries,
                                                overrides=kwargs.get("override_tools"))
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
                        logger.info("%s %s is Running", self.label(j), uuid)
                        j.running = True
                        j.update_pipeline_component(event["properties"]["new_attributes"])
                elif event["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled", "Final"):
                    uuid = event["object_uuid"]
                    try:
                        self.cond.acquire()
                        j = self.processes[uuid]
                        logger.info("%s %s is %s", self.label(j), uuid, event["properties"]["new_attributes"]["state"])
                        with Perf(metrics, "done %s" % j.name):
                            j.done(event["properties"]["new_attributes"])
                        self.cond.notify()
                    finally:
                        self.cond.release()

    def label(self, obj):
        return "[%s %s]" % (self.work_api[0:-1], obj.name)

    def poll_states(self):
        """Poll status of jobs or containers listed in the processes dict.

        Runs in a separate thread.
        """

        try:
            while True:
                self.stop_polling.wait(15)
                if self.stop_polling.is_set():
                    break
                with self.lock:
                    keys = self.processes.keys()
                if not keys:
                    continue

                if self.work_api == "containers":
                    table = self.poll_api.container_requests()
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
        except:
            logger.error("Fatal error in state polling thread.", exc_info=(sys.exc_info()[1] if self.debug else False))
            self.cond.acquire()
            self.processes.clear()
            self.cond.notify()
            self.cond.release()
        finally:
            self.stop_polling.set()

    def get_uploaded(self):
        return self.uploaded.copy()

    def add_uploaded(self, src, pair):
        self.uploaded[src] = pair

    def add_intermediate_output(self, uuid):
        if uuid:
            self.intermediate_output_collections.append(uuid)

    def trash_intermediate_output(self):
        logger.info("Cleaning up intermediate output collections")
        for i in self.intermediate_output_collections:
            try:
                self.api.collections().delete(uuid=i).execute(num_retries=self.num_retries)
            except:
                logger.warn("Failed to delete intermediate output: %s", sys.exc_info()[1], exc_info=(sys.exc_info()[1] if self.debug else False))
            if sys.exc_info()[0] is KeyboardInterrupt:
                break

    def check_features(self, obj):
        if isinstance(obj, dict):
            if obj.get("writable"):
                raise SourceLine(obj, "writable", UnsupportedRequirement).makeError("InitialWorkDir feature 'writable: true' not supported")
            if obj.get("class") == "DockerRequirement":
                if obj.get("dockerOutputDirectory"):
                    # TODO: can be supported by containers API, but not jobs API.
                    raise SourceLine(obj, "dockerOutputDirectory", UnsupportedRequirement).makeError(
                        "Option 'dockerOutputDirectory' of DockerRequirement not supported.")
            for v in obj.itervalues():
                self.check_features(v)
        elif isinstance(obj, list):
            for i,v in enumerate(obj):
                with SourceLine(obj, i, UnsupportedRequirement):
                    self.check_features(v)

    def make_output_collection(self, name, tagsString, outputObj):
        outputObj = copy.deepcopy(outputObj)

        files = []
        def capture(fileobj):
            files.append(fileobj)

        adjustDirObjs(outputObj, capture)
        adjustFileObjs(outputObj, capture)

        generatemapper = NoFollowPathMapper(files, "", "", separateDirs=False)

        final = arvados.collection.Collection(api_client=self.api,
                                              keep_client=self.keep_client,
                                              num_retries=self.num_retries)

        for k,v in generatemapper.items():
            if k.startswith("_:"):
                if v.type == "Directory":
                    continue
                if v.type == "CreateFile":
                    with final.open(v.target, "wb") as f:
                        f.write(v.resolved.encode("utf-8"))
                    continue

            if not k.startswith("keep:"):
                raise Exception("Output source is not in keep or a literal")
            sp = k.split("/")
            srccollection = sp[0][5:]
            try:
                reader = self.collection_cache.get(srccollection)
                srcpath = "/".join(sp[1:]) if len(sp) > 1 else "."
                final.copy(srcpath, v.target, source_collection=reader, overwrite=False)
            except arvados.errors.ArgumentError as e:
                logger.error("Creating CollectionReader for '%s' '%s': %s", k, v, e)
                raise
            except IOError as e:
                logger.warn("While preparing output collection: %s", e)

        def rewrite(fileobj):
            fileobj["location"] = generatemapper.mapper(fileobj["location"]).target
            for k in ("basename", "listing", "contents"):
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

        final_uuid = final.manifest_locator()
        tags = tagsString.split(',')
        for tag in tags:
             self.api.links().create(body={
                "head_uuid": final_uuid, "link_class": "tag", "name": tag
                }).execute(num_retries=self.num_retries)

        def finalcollection(fileobj):
            fileobj["location"] = "keep:%s/%s" % (final.portable_data_hash(), fileobj["location"])

        adjustDirObjs(outputObj, finalcollection)
        adjustFileObjs(outputObj, finalcollection)

        return (outputObj, final)

    def set_crunch_output(self):
        if self.work_api == "containers":
            try:
                current = self.api.containers().current().execute(num_retries=self.num_retries)
            except ApiError as e:
                # Status code 404 just means we're not running in a container.
                if e.resp.status != 404:
                    logger.info("Getting current container: %s", e)
                return
            try:
                self.api.containers().update(uuid=current['uuid'],
                                             body={
                                                 'output': self.final_output_collection.portable_data_hash(),
                                             }).execute(num_retries=self.num_retries)
                self.api.collections().update(uuid=self.final_output_collection.manifest_locator(),
                                              body={
                                                  'is_trashed': True
                                              }).execute(num_retries=self.num_retries)
            except Exception as e:
                logger.info("Setting container output: %s", e)
        elif self.work_api == "jobs" and "TASK_UUID" in os.environ:
            self.api.job_tasks().update(uuid=os.environ["TASK_UUID"],
                                   body={
                                       'output': self.final_output_collection.portable_data_hash(),
                                       'success': self.final_status == "success",
                                       'progress':1.0
                                   }).execute(num_retries=self.num_retries)

    def arv_executor(self, tool, job_order, **kwargs):
        self.debug = kwargs.get("debug")

        tool.visit(self.check_features)

        self.project_uuid = kwargs.get("project_uuid")
        self.pipeline = None
        make_fs_access = kwargs.get("make_fs_access") or partial(CollectionFsAccess,
                                                                 collection_cache=self.collection_cache)
        self.fs_access = make_fs_access(kwargs["basedir"])


        self.trash_intermediate = kwargs["trash_intermediate"]
        if self.trash_intermediate and self.work_api != "containers":
            raise Exception("--trash-intermediate is only supported with --api=containers.")

        self.intermediate_output_ttl = kwargs["intermediate_output_ttl"]
        if self.intermediate_output_ttl and self.work_api != "containers":
            raise Exception("--intermediate-output-ttl is only supported with --api=containers.")
        if self.intermediate_output_ttl < 0:
            raise Exception("Invalid value %d for --intermediate-output-ttl, cannot be less than zero" % self.intermediate_output_ttl)

        if not kwargs.get("name"):
            kwargs["name"] = self.name = tool.tool.get("label") or tool.metadata.get("label") or os.path.basename(tool.tool["id"])

        # Upload direct dependencies of workflow steps, get back mapping of files to keep references.
        # Also uploads docker images.
        override_tools = {}
        upload_workflow_deps(self, tool, override_tools)

        # Reload tool object which may have been updated by
        # upload_workflow_deps
        tool = self.arv_make_tool(tool.doc_loader.idx[tool.tool["id"]],
                                  makeTool=self.arv_make_tool,
                                  loader=tool.doc_loader,
                                  avsc_names=tool.doc_schema,
                                  metadata=tool.metadata,
                                  override_tools=override_tools)

        # Upload local file references in the job order.
        job_order = upload_job_order(self, "%s input" % kwargs["name"],
                                     tool, job_order)

        existing_uuid = kwargs.get("update_workflow")
        if existing_uuid or kwargs.get("create_workflow"):
            # Create a pipeline template or workflow record and exit.
            if self.work_api == "jobs":
                tmpl = RunnerTemplate(self, tool, job_order,
                                      kwargs.get("enable_reuse"),
                                      uuid=existing_uuid,
                                      submit_runner_ram=kwargs.get("submit_runner_ram"),
                                      name=kwargs["name"])
                tmpl.save()
                # cwltool.main will write our return value to stdout.
                return (tmpl.uuid, "success")
            elif self.work_api == "containers":
                return (upload_workflow(self, tool, job_order,
                                        self.project_uuid,
                                        uuid=existing_uuid,
                                        submit_runner_ram=kwargs.get("submit_runner_ram"),
                                        name=kwargs["name"]),
                        "success")

        self.ignore_docker_for_reuse = kwargs.get("ignore_docker_for_reuse")

        kwargs["make_fs_access"] = make_fs_access
        kwargs["enable_reuse"] = kwargs.get("enable_reuse")
        kwargs["use_container"] = True
        kwargs["tmpdir_prefix"] = "tmp"
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
            # Submit a runner job to run the workflow for us.
            if self.work_api == "containers":
                if tool.tool["class"] == "CommandLineTool":
                    kwargs["runnerjob"] = tool.tool["id"]
                    upload_dependencies(self,
                                        kwargs["name"],
                                        tool.doc_loader,
                                        tool.tool,
                                        tool.tool["id"],
                                        False)
                    runnerjob = tool.job(job_order,
                                         self.output_callback,
                                         **kwargs).next()
                else:
                    runnerjob = RunnerContainer(self, tool, job_order, kwargs.get("enable_reuse"),
                                                self.output_name,
                                                self.output_tags,
                                                submit_runner_ram=kwargs.get("submit_runner_ram"),
                                                name=kwargs.get("name"),
                                                on_error=kwargs.get("on_error"),
                                                submit_runner_image=kwargs.get("submit_runner_image"),
                                                intermediate_output_ttl=kwargs.get("intermediate_output_ttl"))
            elif self.work_api == "jobs":
                runnerjob = RunnerJob(self, tool, job_order, kwargs.get("enable_reuse"),
                                      self.output_name,
                                      self.output_tags,
                                      submit_runner_ram=kwargs.get("submit_runner_ram"),
                                      name=kwargs.get("name"),
                                      on_error=kwargs.get("on_error"),
                                      submit_runner_image=kwargs.get("submit_runner_image"))

        if not kwargs.get("submit") and "cwl_runner_job" not in kwargs and self.work_api == "jobs":
            # Create pipeline for local run
            self.pipeline = self.api.pipeline_instances().create(
                body={
                    "owner_uuid": self.project_uuid,
                    "name": kwargs["name"] if kwargs.get("name") else shortname(tool.tool["id"]),
                    "components": {},
                    "state": "RunningOnClient"}).execute(num_retries=self.num_retries)
            logger.info("Pipeline instance %s", self.pipeline["uuid"])

        if runnerjob and not kwargs.get("wait"):
            runnerjob.run(wait=kwargs.get("wait"))
            return (runnerjob.uuid, "success")

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

                if self.stop_polling.is_set():
                    break

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
                logger.error("Execution failed: %s", sys.exc_info()[1], exc_info=(sys.exc_info()[1] if self.debug else False))
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

        if self.final_output is None:
            raise WorkflowException("Workflow did not return a result.")

        if kwargs.get("submit") and isinstance(runnerjob, Runner):
            logger.info("Final output collection %s", runnerjob.final_output)
        else:
            if self.output_name is None:
                self.output_name = "Output of %s" % (shortname(tool.tool["id"]))
            if self.output_tags is None:
                self.output_tags = ""
            self.final_output, self.final_output_collection = self.make_output_collection(self.output_name, self.output_tags, self.final_output)
            self.set_crunch_output()

        if kwargs.get("compute_checksum"):
            adjustDirObjs(self.final_output, partial(get_listing, self.fs_access))
            adjustFileObjs(self.final_output, partial(compute_checksums, self.fs_access))

        if self.trash_intermediate and self.final_status == "success":
            self.trash_intermediate_output()

        return (self.final_output, self.final_status)


def versionstring():
    """Print version string of key packages for provenance and debugging."""

    arvcwlpkg = pkg_resources.require("arvados-cwl-runner")
    arvpkg = pkg_resources.require("arvados-python-client")
    cwlpkg = pkg_resources.require("cwltool")

    return "%s %s %s, %s %s, %s %s" % (sys.argv[0], __version__, arvcwlpkg[0].version,
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
    exgroup.add_argument("--version", action="store_true", help="Print version and exit")
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
                        default=1024)

    parser.add_argument("--submit-runner-image", type=str,
                        help="Docker image for workflow runner job, default arvados/jobs:%s" % __version__,
                        default=None)

    parser.add_argument("--name", type=str,
                        help="Name to use for workflow execution instance.",
                        default=None)

    parser.add_argument("--on-error", type=str,
                        help="Desired workflow behavior when a step fails.  One of 'stop' or 'continue'. "
                        "Default is 'continue'.", default="continue", choices=("stop", "continue"))

    parser.add_argument("--enable-dev", action="store_true",
                        help="Enable loading and running development versions "
                             "of CWL spec.", default=False)

    parser.add_argument("--intermediate-output-ttl", type=int, metavar="N",
                        help="If N > 0, intermediate output collections will be trashed N seconds after creation.  Default is 0 (don't trash).",
                        default=0)

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--trash-intermediate", action="store_true",
                        default=False, dest="trash_intermediate",
                         help="Immediately trash intermediate outputs on workflow success.")
    exgroup.add_argument("--no-trash-intermediate", action="store_false",
                        default=False, dest="trash_intermediate",
                        help="Do not trash intermediate outputs (default).")

    parser.add_argument("workflow", type=str, nargs="?", default=None, help="The workflow to execute")
    parser.add_argument("job_order", nargs=argparse.REMAINDER, help="The input object to the workflow.")

    return parser

def add_arv_hints():
    cwltool.draft2tool.ACCEPTLIST_EN_RELAXED_RE = re.compile(r".*")
    cwltool.draft2tool.ACCEPTLIST_RE = cwltool.draft2tool.ACCEPTLIST_EN_RELAXED_RE
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
        "http://arvados.org/cwl#IntermediateOutput"
    ])

def main(args, stdout, stderr, api_client=None, keep_client=None):
    parser = arg_parser()

    job_order_object = None
    arvargs = parser.parse_args(args)

    if arvargs.version:
        print versionstring()
        return

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

    try:
        if api_client is None:
            api_client=arvados.api('v1', model=OrderedJsonModel())
        if keep_client is None:
            keep_client = arvados.keep.KeepClient(api_client=api_client, num_retries=4)
        runner = ArvCwlRunner(api_client, work_api=arvargs.work_api, keep_client=keep_client,
                              num_retries=4, output_name=arvargs.output_name,
                              output_tags=arvargs.output_tags)
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

    arvargs.conformance_test = None
    arvargs.use_container = True
    arvargs.relax_path_checks = True
    arvargs.validate = None

    make_fs_access = partial(CollectionFsAccess,
                           collection_cache=runner.collection_cache)

    return cwltool.main.main(args=arvargs,
                             stdout=stdout,
                             stderr=stderr,
                             executor=runner.arv_executor,
                             makeTool=runner.arv_make_tool,
                             versionfunc=versionstring,
                             job_order_object=job_order_object,
                             make_fs_access=make_fs_access,
                             fetcher_constructor=partial(CollectionFetcher,
                                                         api_client=api_client,
                                                         fs_access=make_fs_access(""),
                                                         num_retries=runner.num_retries),
                             resolver=partial(collectionResolver, api_client, num_retries=runner.num_retries),
                             logger_handler=arvados.log_handler,
                             custom_schema_callback=add_arv_hints)
