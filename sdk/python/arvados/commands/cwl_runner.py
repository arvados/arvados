#!/usr/bin/env python

import argparse
import arvados
import arvados.events
import arvados.commands.keepdocker
import arvados.commands.run
import cwltool.draft2tool
import cwltool.workflow
import cwltool.main
import threading
import cwltool.docker
import fnmatch
import logging
import re
import os
from cwltool.process import get_feature

logger = logging.getLogger('arvados.cwl-runner')
logger.setLevel(logging.INFO)

del cwltool.draft2tool.supportedProcessRequirements[cwltool.draft2tool.supportedProcessRequirements.index("EnvVarRequirement")]
del cwltool.draft2tool.supportedProcessRequirements[cwltool.draft2tool.supportedProcessRequirements.index("CreateFileRequirement")]

def arv_docker_get_image(api_client, dockerRequirement, pull_image):
    if "dockerImageId" not in dockerRequirement and "dockerPull" in dockerRequirement:
        dockerRequirement["dockerImageId"] = dockerRequirement["dockerPull"]

    sp = dockerRequirement["dockerImageId"].split(":")
    image_name = sp[0]
    image_tag = sp[1] if len(sp) > 1 else None

    images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                            image_name=image_name,
                                                            image_tag=image_tag)

    if not images:
        imageId = cwltool.docker.get_image(dockerRequirement, pull_image)
        args = [image_name]
        if image_tag:
            args.append(image_tag)
        arvados.commands.keepdocker.main(args)

    return dockerRequirement["dockerImageId"]

class CollectionFsAccess(object):
    def __init__(self):
        self.collections = {}

    def get_collection(self, path):
        p = path.split("/")
        if p[0] == "keep":
            del p[0]
        if p[0] not in self.collections:
            self.collections[p[0]] = arvados.collection.CollectionReader(p[0])
        return (self.collections[p[0]], "/".join(p[1:]))

    def _match(self, collection, patternsegments, parent):
        ret = []
        for i in collection:
            if fnmatch.fnmatch(i, patternsegments[0]):
                cur = os.path.join(parent, i)
                if len(patternsegments) == 1:
                    ret.append(cur)
                else:
                    ret.extend(self._match(collection[i], patternsegments[1:], cur))
        return ret

    def glob(self, pattern):
        collection, rest = self.get_collection(pattern)
        patternsegments = rest.split("/")
        return self._match(collection, patternsegments, collection.manifest_locator())

    def open(self, fn, mode):
        collection, rest = self.get_collection(fn)
        return collection.open(rest, mode)

    def exists(self, fn):
        collection, rest = self.get_collection(fn)
        return collection.exists(rest)


class ArvadosJob(object):
    def __init__(self, runner):
        self.arvrunner = runner
        self.running = False

    def run(self, dry_run=False, pull_image=True, **kwargs):
        script_parameters = {
            "command": self.command_line
        }
        runtime_constraints = {}

        if self.stdin:
            script_parameters["task.stdin"] = self.pathmapper.mapper(self.stdin)[1]

        if self.stdout:
            script_parameters["task.stdout"] = self.stdout

        (docker_req, docker_is_req) = get_feature(self, "DockerRequirement")
        if docker_req and kwargs.get("use_container") is not False:
            runtime_constraints["docker_image"] = arv_docker_get_image(self.arvrunner.api, docker_req, pull_image)
            runtime_constraints["arvados_sdk_version"] = "master"

        response = self.arvrunner.api.jobs().create(body={
            "script": "run-command",
            "repository": "arvados",
            "script_version": "master",
            "script_parameters": script_parameters,
            "runtime_constraints": runtime_constraints
        }).execute()

        logger.info("Submitted job %s", response["uuid"])

        self.arvrunner.jobs[response["uuid"]] = self

    def done(self, record):
        try:
            if record["state"] == "Complete":
                processStatus = "success"
            else:
                processStatus = "permanentFail"

            try:
                outputs = {}
                outputs = self.collect_outputs(record["output"])
            except Exception as e:
                logger.warn(str(e))
                processStatus = "permanentFail"

            self.output_callback(outputs, processStatus)
        finally:
            del self.arvrunner.jobs[record["uuid"]]

class ArvPathMapper(cwltool.pathmapper.PathMapper):
    def __init__(self, arvrunner, referenced_files, basedir, **kwargs):
        self._pathmap = {}
        uploadfiles = []

        pdh_path = re.compile(r'^[0-9a-f]{32}\+\d+/(.*)')

        for src in referenced_files:
            ab = src if os.path.isabs(src) else os.path.join(basedir, src)
            st = arvados.commands.run.statfile("", ab)
            if kwargs.get("conformance_test"):
                self._pathmap[src] = (src, ab)
            elif isinstance(st, arvados.commands.run.UploadFile):
                uploadfiles.append((src, ab, st))
            elif isinstance(st, arvados.commands.run.ArvFile):
                self._pathmap[src] = (ab, st.fn)
            elif isinstance(st, basestring) and pdh_path.match(st):
                self._pathmap[src] = (st, "$(file %s)" % st)
            else:
                raise cwltool.workflow.WorkflowException("Input file path '%s' is invalid", st)

        if uploadfiles:
            arvados.commands.run.uploadfiles([u[2] for u in uploadfiles], arvrunner.api, dry_run=kwargs.get("dry_run"), num_retries=3)

        for src, ab, st in uploadfiles:
            self._pathmap[src] = (ab, st.fn)



class ArvadosCommandTool(cwltool.draft2tool.CommandLineTool):
    def __init__(self, arvrunner, toolpath_object, **kwargs):
        super(ArvadosCommandTool, self).__init__(toolpath_object, **kwargs)
        self.arvrunner = arvrunner

    def makeJobRunner(self):
        return ArvadosJob(self.arvrunner)

    def makePathMapper(self, reffiles, input_basedir, **kwargs):
        return ArvPathMapper(self.arvrunner, reffiles, input_basedir, **kwargs)


class ArvCwlRunner(object):
    def __init__(self, api_client):
        self.api = api_client
        self.jobs = {}
        self.lock = threading.Lock()
        self.cond = threading.Condition(self.lock)
        self.final_output = None

    def arvMakeTool(self, toolpath_object, **kwargs):
        if "class" in toolpath_object and toolpath_object["class"] == "CommandLineTool":
            return ArvadosCommandTool(self, toolpath_object, **kwargs)
        else:
            return cwltool.workflow.defaultMakeTool(toolpath_object, **kwargs)

    def output_callback(self, out, processStatus):
        if processStatus == "success":
            logger.info("Overall job status is %s", processStatus)
        else:
            logger.warn("Overall job status is %s", processStatus)
        self.final_output = out

    def on_message(self, event):
        if "object_uuid" in event:
                if event["object_uuid"] in self.jobs and event["event_type"] == "update":
                    if event["properties"]["new_attributes"]["state"] == "Running" and self.jobs[event["object_uuid"]].running is False:
                        logger.info("Job %s is Running", event["object_uuid"])
                        self.jobs[event["object_uuid"]].running = True
                    elif event["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled"):
                        logger.info("Job %s is %s", event["object_uuid"], event["properties"]["new_attributes"]["state"])
                        try:
                            self.cond.acquire()
                            self.jobs[event["object_uuid"]].done(event["properties"]["new_attributes"])
                            self.cond.notify()
                        finally:
                            self.cond.release()

    def arvExecutor(self, t, job_order, input_basedir, **kwargs):
        events = arvados.events.subscribe(arvados.api('v1'), [["object_uuid", "is_a", "arvados#job"]], self.on_message)

        kwargs["fs_access"] = CollectionFsAccess()

        if kwargs.get("conformance_test"):
            return cwltool.main.single_job_executor(t, job_order, input_basedir, **kwargs)
        else:
            jobiter = t.job(job_order,
                            input_basedir,
                            self.output_callback,
                            **kwargs)

            for r in jobiter:
                if r:
                    with self.lock:
                        r.run(**kwargs)
                else:
                    if self.jobs:
                        try:
                            self.cond.acquire()
                            self.cond.wait()
                        finally:
                            self.cond.release()
                    else:
                        raise cwltool.workflow.WorkflowException("Workflow deadlocked.")

            while self.jobs:
                try:
                    self.cond.acquire()
                    self.cond.wait()
                finally:
                    self.cond.release()

            events.close()

            if self.final_output is None:
                raise cwltool.workflow.WorkflowException("Workflow did not return a result.")

            return self.final_output


def main(args, stdout, stderr, api_client=None):
    runner = ArvCwlRunner(api_client=arvados.api('v1'))
    args.append("--leave-outputs")
    return cwltool.main.main(args, executor=runner.arvExecutor, makeTool=runner.arvMakeTool)
