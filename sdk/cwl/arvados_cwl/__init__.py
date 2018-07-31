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
import threading
import hashlib
import copy
import json
import re
from functools import partial
import pkg_resources  # part of setuptools
import Queue
import time
import signal
import thread

from cwltool.errors import WorkflowException
import cwltool.main
import cwltool.workflow
import cwltool.process
from schema_salad.sourceline import SourceLine
import schema_salad.validate as validate
import cwltool.argparser

import arvados
import arvados.config
from arvados.keep import KeepClient
from arvados.errors import ApiError
import arvados.commands._util as arv_cmd

from .arvcontainer import ArvadosContainer, RunnerContainer
from .arvjob import ArvadosJob, RunnerJob, RunnerTemplate
from. runner import Runner, upload_docker, upload_job_order, upload_workflow_deps
from .arvtool import ArvadosCommandTool
from .arvworkflow import ArvadosWorkflow, upload_workflow
from .fsaccess import CollectionFsAccess, CollectionFetcher, collectionResolver, CollectionCache
from .perf import Perf
from .pathmapper import NoFollowPathMapper
from .task_queue import TaskQueue
from .context import ArvLoadingContext, ArvRuntimeContext
from ._version import __version__

from cwltool.pack import pack
from cwltool.process import shortname, UnsupportedRequirement, use_custom_schema
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs, get_listing
from cwltool.command_line_tool import compute_checksums

from arvados.api import OrderedJsonModel

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')
logger.setLevel(logging.INFO)

arvados.log_handler.setFormatter(logging.Formatter(
        '%(asctime)s %(name)s %(levelname)s: %(message)s',
        '%Y-%m-%d %H:%M:%S'))

DEFAULT_PRIORITY = 500

class ArvCwlRunner(object):
    """Execute a CWL tool or workflow, submit work (using either jobs or
    containers API), wait for them to complete, and report output.

    """

    def __init__(self, api_client,
                 arvargs=None,
                 keep_client=None,
                 num_retries=4,
                 thread_count=4):

        if arvargs is None:
            arvargs = argparse.Namespace()
            arvargs.work_api = None
            arvargs.output_name = None
            arvargs.output_tags = None
            arvargs.thread_count = 1

        self.api = api_client
        self.processes = {}
        self.workflow_eval_lock = threading.Condition(threading.RLock())
        self.final_output = None
        self.final_status = None
        self.num_retries = num_retries
        self.uuid = None
        self.stop_polling = threading.Event()
        self.poll_api = None
        self.pipeline = None
        self.final_output_collection = None
        self.output_name = arvargs.output_name
        self.output_tags = arvargs.output_tags
        self.project_uuid = None
        self.intermediate_output_ttl = 0
        self.intermediate_output_collections = []
        self.trash_intermediate = False
        self.thread_count = arvargs.thread_count
        self.poll_interval = 12
        self.loadingContext = None

        if keep_client is not None:
            self.keep_client = keep_client
        else:
            self.keep_client = arvados.keep.KeepClient(api_client=self.api, num_retries=self.num_retries)

        self.collection_cache = CollectionCache(self.api, self.keep_client, self.num_retries)

        self.fetcher_constructor = partial(CollectionFetcher,
                                           api_client=self.api,
                                           fs_access=CollectionFsAccess("", collection_cache=self.collection_cache),
                                           num_retries=self.num_retries)

        self.work_api = None
        expected_api = ["jobs", "containers"]
        for api in expected_api:
            try:
                methods = self.api._rootDesc.get('resources')[api]['methods']
                if ('httpMethod' in methods['create'] and
                    (arvargs.work_api == api or arvargs.work_api is None)):
                    self.work_api = api
                    break
            except KeyError:
                pass

        if not self.work_api:
            if arvargs.work_api is None:
                raise Exception("No supported APIs")
            else:
                raise Exception("Unsupported API '%s', expected one of %s" % (work_api, expected_api))

        if self.work_api == "jobs":
            logger.warn("""
*******************************
Using the deprecated 'jobs' API.

To get rid of this warning:

Users: read about migrating at
http://doc.arvados.org/user/cwl/cwl-style.html#migrate
and use the option --api=containers

Admins: configure the cluster to disable the 'jobs' API as described at:
http://doc.arvados.org/install/install-api-server.html#disable_api_methods
*******************************""")

        self.loadingContext = ArvLoadingContext(vars(arvargs))
        self.loadingContext.fetcher_constructor = self.fetcher_constructor
        self.loadingContext.resolver = partial(collectionResolver, self.api, num_retries=self.num_retries)
        self.loadingContext.construct_tool_object = self.arv_make_tool


    def arv_make_tool(self, toolpath_object, loadingContext):
        if "class" in toolpath_object and toolpath_object["class"] == "CommandLineTool":
            return ArvadosCommandTool(self, toolpath_object, loadingContext)
        elif "class" in toolpath_object and toolpath_object["class"] == "Workflow":
            return ArvadosWorkflow(self, toolpath_object, loadingContext)
        else:
            return cwltool.workflow.default_make_tool(toolpath_object, loadingContext)

    def output_callback(self, out, processStatus):
        with self.workflow_eval_lock:
            if processStatus == "success":
                logger.info("Overall process status is %s", processStatus)
                if self.pipeline:
                    self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                         body={"state": "Complete"}).execute(num_retries=self.num_retries)
            else:
                logger.error("Overall process status is %s", processStatus)
                if self.pipeline:
                    self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                         body={"state": "Failed"}).execute(num_retries=self.num_retries)
            self.final_status = processStatus
            self.final_output = out
            self.workflow_eval_lock.notifyAll()


    def start_run(self, runnable, runtimeContext):
        self.task_queue.add(partial(runnable.run, runtimeContext))

    def process_submitted(self, container):
        with self.workflow_eval_lock:
            self.processes[container.uuid] = container

    def process_done(self, uuid, record):
        with self.workflow_eval_lock:
            j = self.processes[uuid]
            logger.info("%s %s is %s", self.label(j), uuid, record["state"])
            self.task_queue.add(partial(j.done, record))
            del self.processes[uuid]

    def wrapped_callback(self, cb, obj, st):
        with self.workflow_eval_lock:
            cb(obj, st)
            self.workflow_eval_lock.notifyAll()

    def get_wrapped_callback(self, cb):
        return partial(self.wrapped_callback, cb)

    def on_message(self, event):
        if event.get("object_uuid") in self.processes and event["event_type"] == "update":
            uuid = event["object_uuid"]
            if event["properties"]["new_attributes"]["state"] == "Running":
                with self.workflow_eval_lock:
                    j = self.processes[uuid]
                    if j.running is False:
                        j.running = True
                        j.update_pipeline_component(event["properties"]["new_attributes"])
                        logger.info("%s %s is Running", self.label(j), uuid)
            elif event["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled", "Final"):
                self.process_done(uuid, event["properties"]["new_attributes"])

    def label(self, obj):
        return "[%s %s]" % (self.work_api[0:-1], obj.name)

    def poll_states(self):
        """Poll status of jobs or containers listed in the processes dict.

        Runs in a separate thread.
        """

        try:
            remain_wait = self.poll_interval
            while True:
                if remain_wait > 0:
                    self.stop_polling.wait(remain_wait)
                if self.stop_polling.is_set():
                    break
                with self.workflow_eval_lock:
                    keys = list(self.processes.keys())
                if not keys:
                    remain_wait = self.poll_interval
                    continue

                begin_poll = time.time()
                if self.work_api == "containers":
                    table = self.poll_api.container_requests()
                elif self.work_api == "jobs":
                    table = self.poll_api.jobs()

                try:
                    proc_states = table.list(filters=[["uuid", "in", keys]]).execute(num_retries=self.num_retries)
                except Exception as e:
                    logger.warn("Error checking states on API server: %s", e)
                    remain_wait = self.poll_interval
                    continue

                for p in proc_states["items"]:
                    self.on_message({
                        "object_uuid": p["uuid"],
                        "event_type": "update",
                        "properties": {
                            "new_attributes": p
                        }
                    })
                finish_poll = time.time()
                remain_wait = self.poll_interval - (finish_poll - begin_poll)
        except:
            logger.exception("Fatal error in state polling thread.")
            with self.workflow_eval_lock:
                self.processes.clear()
                self.workflow_eval_lock.notifyAll()
        finally:
            self.stop_polling.set()

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
            if sys.exc_info()[0] is KeyboardInterrupt or sys.exc_info()[0] is SystemExit:
                break

    def check_features(self, obj):
        if isinstance(obj, dict):
            if obj.get("writable") and self.work_api != "containers":
                raise SourceLine(obj, "writable", UnsupportedRequirement).makeError("InitialWorkDir feature 'writable: true' not supported with --api=jobs")
            if obj.get("class") == "DockerRequirement":
                if obj.get("dockerOutputDirectory"):
                    if self.work_api != "containers":
                        raise SourceLine(obj, "dockerOutputDirectory", UnsupportedRequirement).makeError(
                            "Option 'dockerOutputDirectory' of DockerRequirement not supported with --api=jobs.")
                    if not obj.get("dockerOutputDirectory").startswith('/'):
                        raise SourceLine(obj, "dockerOutputDirectory", validate.ValidationException).makeError(
                            "Option 'dockerOutputDirectory' must be an absolute path.")
            if obj.get("class") == "http://commonwl.org/cwltool#Secrets" and self.work_api != "containers":
                raise SourceLine(obj, "class", UnsupportedRequirement).makeError("Secrets not supported with --api=jobs")
            for v in obj.itervalues():
                self.check_features(v)
        elif isinstance(obj, list):
            for i,v in enumerate(obj):
                with SourceLine(obj, i, UnsupportedRequirement, logger.isEnabledFor(logging.DEBUG)):
                    self.check_features(v)

    def make_output_collection(self, name, storage_classes, tagsString, outputObj):
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
            for k in ("listing", "contents", "nameext", "nameroot", "dirname"):
                if k in fileobj:
                    del fileobj[k]

        adjustDirObjs(outputObj, rewrite)
        adjustFileObjs(outputObj, rewrite)

        with final.open("cwl.output.json", "w") as f:
            json.dump(outputObj, f, sort_keys=True, indent=4, separators=(',',': '))

        final.save_new(name=name, owner_uuid=self.project_uuid, storage_classes=storage_classes, ensure_unique_name=True)

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

    def arv_executor(self, tool, job_order, runtimeContext, logger=None):
        self.debug = runtimeContext.debug

        tool.visit(self.check_features)

        self.project_uuid = runtimeContext.project_uuid
        self.pipeline = None
        self.fs_access = runtimeContext.make_fs_access(runtimeContext.basedir)
        self.secret_store = runtimeContext.secret_store

        self.trash_intermediate = runtimeContext.trash_intermediate
        if self.trash_intermediate and self.work_api != "containers":
            raise Exception("--trash-intermediate is only supported with --api=containers.")

        self.intermediate_output_ttl = runtimeContext.intermediate_output_ttl
        if self.intermediate_output_ttl and self.work_api != "containers":
            raise Exception("--intermediate-output-ttl is only supported with --api=containers.")
        if self.intermediate_output_ttl < 0:
            raise Exception("Invalid value %d for --intermediate-output-ttl, cannot be less than zero" % self.intermediate_output_ttl)

        if runtimeContext.submit_request_uuid and self.work_api != "containers":
            raise Exception("--submit-request-uuid requires containers API, but using '{}' api".format(self.work_api))

        if not runtimeContext.name:
            runtimeContext.name = self.name = tool.tool.get("label") or tool.metadata.get("label") or os.path.basename(tool.tool["id"])

        # Upload direct dependencies of workflow steps, get back mapping of files to keep references.
        # Also uploads docker images.
        merged_map = upload_workflow_deps(self, tool)

        # Reload tool object which may have been updated by
        # upload_workflow_deps
        # Don't validate this time because it will just print redundant errors.
        loadingContext = self.loadingContext.copy()
        loadingContext.loader = tool.doc_loader
        loadingContext.avsc_names = tool.doc_schema
        loadingContext.metadata = tool.metadata
        loadingContext.do_validate = False

        tool = self.arv_make_tool(tool.doc_loader.idx[tool.tool["id"]],
                                  loadingContext)

        # Upload local file references in the job order.
        job_order = upload_job_order(self, "%s input" % runtimeContext.name,
                                     tool, job_order)

        existing_uuid = runtimeContext.update_workflow
        if existing_uuid or runtimeContext.create_workflow:
            # Create a pipeline template or workflow record and exit.
            if self.work_api == "jobs":
                tmpl = RunnerTemplate(self, tool, job_order,
                                      runtimeContext.enable_reuse,
                                      uuid=existing_uuid,
                                      submit_runner_ram=runtimeContext.submit_runner_ram,
                                      name=runtimeContext.name,
                                      merged_map=merged_map)
                tmpl.save()
                # cwltool.main will write our return value to stdout.
                return (tmpl.uuid, "success")
            elif self.work_api == "containers":
                return (upload_workflow(self, tool, job_order,
                                        self.project_uuid,
                                        uuid=existing_uuid,
                                        submit_runner_ram=runtimeContext.submit_runner_ram,
                                        name=runtimeContext.name,
                                        merged_map=merged_map),
                        "success")

        self.ignore_docker_for_reuse = runtimeContext.ignore_docker_for_reuse
        self.eval_timeout = runtimeContext.eval_timeout

        runtimeContext = runtimeContext.copy()
        runtimeContext.use_container = True
        runtimeContext.tmpdir_prefix = "tmp"
        runtimeContext.work_api = self.work_api

        if self.work_api == "containers":
            if self.ignore_docker_for_reuse:
                raise Exception("--ignore-docker-for-reuse not supported with containers API.")
            runtimeContext.outdir = "/var/spool/cwl"
            runtimeContext.docker_outdir = "/var/spool/cwl"
            runtimeContext.tmpdir = "/tmp"
            runtimeContext.docker_tmpdir = "/tmp"
        elif self.work_api == "jobs":
            if runtimeContext.priority != DEFAULT_PRIORITY:
                raise Exception("--priority not implemented for jobs API.")
            runtimeContext.outdir = "$(task.outdir)"
            runtimeContext.docker_outdir = "$(task.outdir)"
            runtimeContext.tmpdir = "$(task.tmpdir)"

        if runtimeContext.priority < 1 or runtimeContext.priority > 1000:
            raise Exception("--priority must be in the range 1..1000.")

        runnerjob = None
        if runtimeContext.submit:
            # Submit a runner job to run the workflow for us.
            if self.work_api == "containers":
                if tool.tool["class"] == "CommandLineTool" and runtimeContext.wait:
                    runtimeContext.runnerjob = tool.tool["id"]
                    runnerjob = tool.job(job_order,
                                         self.output_callback,
                                         runtimeContext).next()
                else:
                    runnerjob = RunnerContainer(self, tool, job_order, runtimeContext.enable_reuse,
                                                self.output_name,
                                                self.output_tags,
                                                submit_runner_ram=runtimeContext.submit_runner_ram,
                                                name=runtimeContext.name,
                                                on_error=runtimeContext.on_error,
                                                submit_runner_image=runtimeContext.submit_runner_image,
                                                intermediate_output_ttl=runtimeContext.intermediate_output_ttl,
                                                merged_map=merged_map,
                                                priority=runtimeContext.priority,
                                                secret_store=self.secret_store)
            elif self.work_api == "jobs":
                runnerjob = RunnerJob(self, tool, job_order, runtimeContext.enable_reuse,
                                      self.output_name,
                                      self.output_tags,
                                      submit_runner_ram=runtimeContext.submit_runner_ram,
                                      name=runtimeContext.name,
                                      on_error=runtimeContext.on_error,
                                      submit_runner_image=runtimeContext.submit_runner_image,
                                      merged_map=merged_map)
        elif runtimeContext.cwl_runner_job is None and self.work_api == "jobs":
            # Create pipeline for local run
            self.pipeline = self.api.pipeline_instances().create(
                body={
                    "owner_uuid": self.project_uuid,
                    "name": runtimeContext.name if runtimeContext.name else shortname(tool.tool["id"]),
                    "components": {},
                    "state": "RunningOnClient"}).execute(num_retries=self.num_retries)
            logger.info("Pipeline instance %s", self.pipeline["uuid"])

        if runnerjob and not runtimeContext.wait:
            submitargs = runtimeContext.copy()
            submitargs.submit = False
            runnerjob.run(submitargs)
            return (runnerjob.uuid, "success")

        self.poll_api = arvados.api('v1')
        self.polling_thread = threading.Thread(target=self.poll_states)
        self.polling_thread.start()

        self.task_queue = TaskQueue(self.workflow_eval_lock, self.thread_count)

        if runnerjob:
            jobiter = iter((runnerjob,))
        else:
            if runtimeContext.cwl_runner_job is not None:
                self.uuid = runtimeContext.cwl_runner_job.get('uuid')
            jobiter = tool.job(job_order,
                               self.output_callback,
                               runtimeContext)

        try:
            self.workflow_eval_lock.acquire()
            # Holds the lock while this code runs and releases it when
            # it is safe to do so in self.workflow_eval_lock.wait(),
            # at which point on_message can update job state and
            # process output callbacks.

            loopperf = Perf(metrics, "jobiter")
            loopperf.__enter__()
            for runnable in jobiter:
                loopperf.__exit__()

                if self.stop_polling.is_set():
                    break

                if self.task_queue.error is not None:
                    raise self.task_queue.error

                if runnable:
                    with Perf(metrics, "run"):
                        self.start_run(runnable, runtimeContext)
                else:
                    if (self.task_queue.in_flight + len(self.processes)) > 0:
                        self.workflow_eval_lock.wait(3)
                    else:
                        logger.error("Workflow is deadlocked, no runnable processes and not waiting on any pending processes.")
                        break
                loopperf.__enter__()
            loopperf.__exit__()

            while (self.task_queue.in_flight + len(self.processes)) > 0:
                if self.task_queue.error is not None:
                    raise self.task_queue.error
                self.workflow_eval_lock.wait(3)

        except UnsupportedRequirement:
            raise
        except:
            if sys.exc_info()[0] is KeyboardInterrupt or sys.exc_info()[0] is SystemExit:
                logger.error("Interrupted, workflow will be cancelled")
            else:
                logger.error("Execution failed: %s", sys.exc_info()[1], exc_info=(sys.exc_info()[1] if self.debug else False))
            if self.pipeline:
                self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                     body={"state": "Failed"}).execute(num_retries=self.num_retries)
            if runnerjob and runnerjob.uuid and self.work_api == "containers":
                self.api.container_requests().update(uuid=runnerjob.uuid,
                                                     body={"priority": "0"}).execute(num_retries=self.num_retries)
        finally:
            self.workflow_eval_lock.release()
            self.task_queue.drain()
            self.stop_polling.set()
            self.polling_thread.join()
            self.task_queue.join()

        if self.final_status == "UnsupportedRequirement":
            raise UnsupportedRequirement("Check log for details.")

        if self.final_output is None:
            raise WorkflowException("Workflow did not return a result.")

        if runtimeContext.submit and isinstance(runnerjob, Runner):
            logger.info("Final output collection %s", runnerjob.final_output)
        else:
            if self.output_name is None:
                self.output_name = "Output of %s" % (shortname(tool.tool["id"]))
            if self.output_tags is None:
                self.output_tags = ""

            storage_classes = runtimeContext.storage_classes.strip().split(",")
            self.final_output, self.final_output_collection = self.make_output_collection(self.output_name, storage_classes, self.output_tags, self.final_output)
            self.set_crunch_output()

        if runtimeContext.compute_checksum:
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

    parser.add_argument("--submit-request-uuid", type=str,
                        default=None,
                        help="Update and commit supplied container request instead of creating a new one (containers API only).")

    parser.add_argument("--name", type=str,
                        help="Name to use for workflow execution instance.",
                        default=None)

    parser.add_argument("--on-error", type=str,
                        help="Desired workflow behavior when a step fails.  One of 'stop' or 'continue'. "
                        "Default is 'continue'.", default="continue", choices=("stop", "continue"))

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
                        default=4, help="Number of threads to use for job submit and output collection.")

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
        "http://arvados.org/cwl#ReuseRequirement"
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

    try:
        if api_client is None:
            api_client = arvados.safeapi.ThreadSafeApiCache(api_params={"model": OrderedJsonModel()}, keep_params={"num_retries": 4})
            keep_client = api_client.keep
            # Make an API object now so errors are reported early.
            api_client.users().current().execute()
        if keep_client is None:
            keep_client = arvados.keep.KeepClient(api_client=api_client, num_retries=4)
        runner = ArvCwlRunner(api_client, arvargs, keep_client=keep_client, num_retries=4)
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

    for key, val in cwltool.argparser.get_default_args().items():
        if not hasattr(arvargs, key):
            setattr(arvargs, key, val)

    runtimeContext = ArvRuntimeContext(vars(arvargs))
    runtimeContext.make_fs_access = partial(CollectionFsAccess,
                             collection_cache=runner.collection_cache)

    return cwltool.main.main(args=arvargs,
                             stdout=stdout,
                             stderr=stderr,
                             executor=runner.arv_executor,
                             versionfunc=versionstring,
                             job_order_object=job_order_object,
                             logger_handler=arvados.log_handler,
                             custom_schema_callback=add_arv_hints,
                             loadingContext=runner.loadingContext,
                             runtimeContext=runtimeContext)
