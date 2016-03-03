#!/usr/bin/env python

import argparse
import arvados
import arvados.events
import arvados.commands.keepdocker
import arvados.commands.run
import arvados.collection
import arvados.util
import cwltool.draft2tool
import cwltool.workflow
import cwltool.main
from cwltool.process import shortname
import threading
import cwltool.docker
import fnmatch
import logging
import re
import os
import sys

from cwltool.process import get_feature
from arvados.api import OrderedJsonModel

logger = logging.getLogger('arvados.cwl-runner')
logger.setLevel(logging.INFO)

crunchrunner_pdh = "83db29f08544e1c319572a6bd971088a+140"
crunchrunner_download = "https://cloud.curoverse.com/collections/download/qr1hi-4zz18-n3m1yxd0vx78jic/1i1u2qtq66k1atziv4ocfgsg5nu5tj11n4r6e0bhvjg03rix4m/crunchrunner"
certs_download = "https://cloud.curoverse.com/collections/download/qr1hi-4zz18-n3m1yxd0vx78jic/1i1u2qtq66k1atziv4ocfgsg5nu5tj11n4r6e0bhvjg03rix4m/ca-certificates.crt"

tmpdirre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.tmpdir\)=(.*)")
outdirre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.outdir\)=(.*)")
keepre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.keep\)=(.*)")


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
        logger.info("Uploading Docker image %s", ":".join(args))
        arvados.commands.keepdocker.main(args)

    return dockerRequirement["dockerImageId"]


class CollectionFsAccess(cwltool.process.StdFsAccess):
    def __init__(self, basedir):
        self.collections = {}
        self.basedir = basedir

    def get_collection(self, path):
        p = path.split("/")
        if p[0].startswith("keep:") and arvados.util.keep_locator_pattern.match(p[0][5:]):
            pdh = p[0][5:]
            if pdh not in self.collections:
                self.collections[pdh] = arvados.collection.CollectionReader(pdh)
            return (self.collections[pdh], "/".join(p[1:]))
        else:
            return (None, path)

    def _match(self, collection, patternsegments, parent):
        if not patternsegments:
            return []

        if not isinstance(collection, arvados.collection.RichCollectionBase):
            return []

        ret = []
        # iterate over the files and subcollections in 'collection'
        for filename in collection:
            if patternsegments[0] == '.':
                # Pattern contains something like "./foo" so just shift
                # past the "./"
                ret.extend(self._match(collection, patternsegments[1:], parent))
            elif fnmatch.fnmatch(filename, patternsegments[0]):
                cur = os.path.join(parent, filename)
                if len(patternsegments) == 1:
                    ret.append(cur)
                else:
                    ret.extend(self._match(collection[filename], patternsegments[1:], cur))
        return ret

    def glob(self, pattern):
        collection, rest = self.get_collection(pattern)
        patternsegments = rest.split("/")
        return self._match(collection, patternsegments, "keep:" + collection.manifest_locator())

    def open(self, fn, mode):
        collection, rest = self.get_collection(fn)
        if collection:
            return collection.open(rest, mode)
        else:
            return open(self._abs(fn), mode)

    def exists(self, fn):
        collection, rest = self.get_collection(fn)
        if collection:
            return collection.exists(rest)
        else:
            return os.path.exists(self._abs(fn))

class ArvadosJob(object):
    def __init__(self, runner):
        self.arvrunner = runner
        self.running = False

    def run(self, dry_run=False, pull_image=True, **kwargs):
        script_parameters = {
            "command": self.command_line
        }
        runtime_constraints = {}

        if self.generatefiles:
            vwd = arvados.collection.Collection()
            script_parameters["task.vwd"] = {}
            for t in self.generatefiles:
                if isinstance(self.generatefiles[t], dict):
                    src, rest = self.arvrunner.fs_access.get_collection(self.generatefiles[t]["path"].replace("$(task.keep)/", "keep:"))
                    vwd.copy(rest, t, source_collection=src)
                else:
                    with vwd.open(t, "w") as f:
                        f.write(self.generatefiles[t])
            vwd.save_new()
            for t in self.generatefiles:
                script_parameters["task.vwd"][t] = "$(task.keep)/%s/%s" % (vwd.portable_data_hash(), t)

        script_parameters["task.env"] = {"TMPDIR": "$(task.tmpdir)"}
        if self.environment:
            script_parameters["task.env"].update(self.environment)

        if self.stdin:
            script_parameters["task.stdin"] = self.pathmapper.mapper(self.stdin)[1]

        if self.stdout:
            script_parameters["task.stdout"] = self.stdout

        (docker_req, docker_is_req) = get_feature(self, "DockerRequirement")
        if docker_req and kwargs.get("use_container") is not False:
            runtime_constraints["docker_image"] = arv_docker_get_image(self.arvrunner.api, docker_req, pull_image)

        try:
            response = self.arvrunner.api.jobs().create(body={
                "script": "crunchrunner",
                "repository": "arvados",
                "script_version": "8488-cwl-crunchrunner-collection",
                "script_parameters": {"tasks": [script_parameters], "crunchrunner": crunchrunner_pdh+"/crunchrunner"},
                "runtime_constraints": runtime_constraints
            }, find_or_create=kwargs.get("enable_reuse", True)).execute(num_retries=self.arvrunner.num_retries)

            self.arvrunner.jobs[response["uuid"]] = self

            self.arvrunner.pipeline["components"][self.name] = {"job": response}
            self.arvrunner.pipeline = self.arvrunner.api.pipeline_instances().update(uuid=self.arvrunner.pipeline["uuid"],
                                                                                     body={
                                                                                         "components": self.arvrunner.pipeline["components"]
                                                                                     }).execute(num_retries=self.arvrunner.num_retries)

            logger.info("Job %s (%s) is %s", self.name, response["uuid"], response["state"])

            if response["state"] in ("Complete", "Failed", "Cancelled"):
                self.done(response)
        except Exception as e:
            logger.error("Got error %s" % str(e))
            self.output_callback({}, "permanentFail")

    def update_pipeline_component(self, record):
        self.arvrunner.pipeline["components"][self.name] = {"job": record}
        self.arvrunner.pipeline = self.arvrunner.api.pipeline_instances().update(uuid=self.arvrunner.pipeline["uuid"],
                                                                                 body={
                                                                                    "components": self.arvrunner.pipeline["components"]
                                                                                 }).execute(num_retries=self.arvrunner.num_retries)

    def done(self, record):
        try:
            self.update_pipeline_component(record)
        except:
            pass

        try:
            if record["state"] == "Complete":
                processStatus = "success"
            else:
                processStatus = "permanentFail"

            try:
                outputs = {}
                if record["output"]:
                    logc = arvados.collection.Collection(record["log"])
                    log = logc.open(logc.keys()[0])
                    tmpdir = None
                    outdir = None
                    keepdir = None
                    for l in log.readlines():
                        g = tmpdirre.match(l)
                        if g:
                            tmpdir = g.group(1)
                        g = outdirre.match(l)
                        if g:
                            outdir = g.group(1)
                        g = keepre.match(l)
                        if g:
                            keepdir = g.group(1)
                        if tmpdir and outdir and keepdir:
                            break

                    self.builder.outdir = outdir
                    self.builder.pathmapper.keepdir = keepdir
                    outputs = self.collect_outputs("keep:" + record["output"])
            except Exception as e:
                logger.exception("Got exception while collecting job outputs:")
                processStatus = "permanentFail"

            self.output_callback(outputs, processStatus)
        finally:
            del self.arvrunner.jobs[record["uuid"]]


class ArvPathMapper(cwltool.pathmapper.PathMapper):
    def __init__(self, arvrunner, referenced_files, basedir, **kwargs):
        self._pathmap = arvrunner.get_uploaded()
        uploadfiles = []

        pdh_path = re.compile(r'^keep:[0-9a-f]{32}\+\d+/.+')

        for src in referenced_files:
            if isinstance(src, basestring) and pdh_path.match(src):
                self._pathmap[src] = (src, "$(task.keep)/%s" % src[5:])
            if src not in self._pathmap:
                ab = cwltool.pathmapper.abspath(src, basedir)
                st = arvados.commands.run.statfile("", ab, fnPattern="$(task.keep)/%s/%s")
                if kwargs.get("conformance_test"):
                    self._pathmap[src] = (src, ab)
                elif isinstance(st, arvados.commands.run.UploadFile):
                    uploadfiles.append((src, ab, st))
                elif isinstance(st, arvados.commands.run.ArvFile):
                    self._pathmap[src] = (ab, st.fn)
                else:
                    raise cwltool.workflow.WorkflowException("Input file path '%s' is invalid" % st)

        if uploadfiles:
            arvados.commands.run.uploadfiles([u[2] for u in uploadfiles],
                                             arvrunner.api,
                                             dry_run=kwargs.get("dry_run"),
                                             num_retries=3,
                                             fnPattern="$(task.keep)/%s/%s")

        for src, ab, st in uploadfiles:
            arvrunner.add_uploaded(src, (ab, st.fn))
            self._pathmap[src] = (ab, st.fn)

        self.keepdir = None

    def reversemap(self, target):
        if target.startswith("keep:"):
            return target
        elif self.keepdir and target.startswith(self.keepdir):
            return "keep:" + target[len(self.keepdir)+1:]
        else:
            return super(ArvPathMapper, self).reversemap(target)


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
        self.uploaded = {}
        self.num_retries = 4

    def arvMakeTool(self, toolpath_object, **kwargs):
        if "class" in toolpath_object and toolpath_object["class"] == "CommandLineTool":
            return ArvadosCommandTool(self, toolpath_object, **kwargs)
        else:
            return cwltool.workflow.defaultMakeTool(toolpath_object, **kwargs)

    def output_callback(self, out, processStatus):
        if processStatus == "success":
            logger.info("Overall job status is %s", processStatus)
            self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                 body={"state": "Complete"}).execute(num_retries=self.num_retries)

        else:
            logger.warn("Overall job status is %s", processStatus)
            self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                 body={"state": "Failed"}).execute(num_retries=self.num_retries)
        self.final_output = out


    def on_message(self, event):
        if "object_uuid" in event:
                if event["object_uuid"] in self.jobs and event["event_type"] == "update":
                    if event["properties"]["new_attributes"]["state"] == "Running" and self.jobs[event["object_uuid"]].running is False:
                        uuid = event["object_uuid"]
                        with self.lock:
                            j = self.jobs[uuid]
                            logger.info("Job %s (%s) is Running", j.name, uuid)
                            j.running = True
                            j.update_pipeline_component(event["properties"]["new_attributes"])
                    elif event["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled"):
                        uuid = event["object_uuid"]
                        try:
                            self.cond.acquire()
                            j = self.jobs[uuid]
                            logger.info("Job %s (%s) is %s", j.name, uuid, event["properties"]["new_attributes"]["state"])
                            j.done(event["properties"]["new_attributes"])
                            self.cond.notify()
                        finally:
                            self.cond.release()

    def get_uploaded(self):
        return self.uploaded.copy()

    def add_uploaded(self, src, pair):
        self.uploaded[src] = pair

    def arvExecutor(self, tool, job_order, input_basedir, args, **kwargs):
        events = arvados.events.subscribe(arvados.api('v1'), [["object_uuid", "is_a", "arvados#job"]], self.on_message)

        try:
            self.api.collections().get(uuid=crunchrunner_pdh).execute()
        except arvados.errors.ApiError as e:
            import httplib2
            h = httplib2.Http(ca_certs=arvados.util.ca_certs_path())
            resp, content = h.request(crunchrunner_download, "GET")
            resp2, content2 = h.request(certs_download, "GET")
            with arvados.collection.Collection() as col:
                with col.open("crunchrunner", "w") as f:
                    f.write(content)
                with col.open("ca-certificates.crt", "w") as f:
                    f.write(content2)

                col.save_new("crunchrunner binary", ensure_unique_name=True)

        self.fs_access = CollectionFsAccess(input_basedir)

        kwargs["fs_access"] = self.fs_access
        kwargs["enable_reuse"] = args.enable_reuse

        kwargs["outdir"] = "$(task.outdir)"
        kwargs["tmpdir"] = "$(task.tmpdir)"

        if kwargs.get("conformance_test"):
            return cwltool.main.single_job_executor(tool, job_order, input_basedir, args, **kwargs)
        else:
            self.pipeline = self.api.pipeline_instances().create(body={"name": shortname(tool.tool["id"]),
                                                                   "components": {},
                                                                   "state": "RunningOnClient"}).execute(num_retries=self.num_retries)

            jobiter = tool.job(job_order,
                               input_basedir,
                               self.output_callback,
                               docker_outdir="$(task.outdir)",
                               **kwargs)

            try:
                for runnable in jobiter:
                    if runnable:
                        with self.lock:
                            runnable.run(**kwargs)
                    else:
                        if self.jobs:
                            try:
                                self.cond.acquire()
                                self.cond.wait(1)
                            except RuntimeError:
                                pass
                            finally:
                                self.cond.release()
                        else:
                            logger.error("Workflow cannot make any more progress.")
                            break

                while self.jobs:
                    try:
                        self.cond.acquire()
                        self.cond.wait(1)
                    except RuntimeError:
                        pass
                    finally:
                        self.cond.release()

                events.close()

                if self.final_output is None:
                    raise cwltool.workflow.WorkflowException("Workflow did not return a result.")

            except:
                if sys.exc_info()[0] is not KeyboardInterrupt:
                    logger.exception("Caught unhandled exception, marking pipeline as failed")
                self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                     body={"state": "Failed"}).execute(num_retries=self.num_retries)

            return self.final_output


def main(args, stdout, stderr, api_client=None):
    args.insert(0, "--leave-outputs")
    parser = cwltool.main.arg_parser()
    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--enable-reuse", action="store_true",
                        default=False, dest="enable_reuse",
                        help="")
    exgroup.add_argument("--disable-reuse", action="store_false",
                        default=False, dest="enable_reuse",
                        help="")

    try:
        runner = ArvCwlRunner(api_client=arvados.api('v1', model=OrderedJsonModel()))
    except Exception as e:
        logger.error(e)
        return 1

    return cwltool.main.main(args, executor=runner.arvExecutor, makeTool=runner.arvMakeTool, parser=parser)
