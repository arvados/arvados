# Crunch script integration for running arvados-cwl-runner (importing
# arvados_cwl module) inside a crunch job.
#
# This gets the job record, transforms the script parameters into a valid CWL
# input object, then executes the CWL runner to run the underlying workflow or
# tool.  When the workflow completes, record the output object in an output
# collection for this runner job.

import arvados
import arvados_cwl
import arvados.collection
import arvados.util
import cwltool.main
import logging
import os
import json
import argparse
import re
import functools

from arvados.api import OrderedJsonModel
from cwltool.process import shortname, adjustFileObjs, adjustDirObjs, getListing, normalizeFilesDirs
from cwltool.load_tool import load_tool
from cwltool.errors import WorkflowException

logger = logging.getLogger('arvados.cwl-runner')

def run():
    # Timestamps are added by crunch-job, so don't print redundant timestamps.
    arvados.log_handler.setFormatter(logging.Formatter('%(name)s %(levelname)s: %(message)s'))

    # Print package versions
    logger.info(arvados_cwl.versionstring())

    api = arvados.api("v1")

    arvados_cwl.add_arv_hints()

    runner = None
    try:
        job_order_object = arvados.current_job()['script_parameters']
        toolpath = "file://%s/%s" % (os.environ['TASK_KEEPMOUNT'], job_order_object.pop("cwl:tool"))

        pdh_path = re.compile(r'^[0-9a-f]{32}\+\d+(/.+)?$')

        def keeppath(v):
            if pdh_path.match(v):
                return "keep:%s" % v
            else:
                return v

        def keeppathObj(v):
            v["location"] = keeppath(v["location"])

        for k,v in job_order_object.items():
            if isinstance(v, basestring) and arvados.util.keep_locator_pattern.match(v):
                job_order_object[k] = {
                    "class": "File",
                    "location": "keep:%s" % v
                }

        adjustFileObjs(job_order_object, keeppathObj)
        adjustDirObjs(job_order_object, keeppathObj)
        normalizeFilesDirs(job_order_object)
        adjustDirObjs(job_order_object, functools.partial(getListing, arvados_cwl.fsaccess.CollectionFsAccess("", api_client=api)))

        output_name = None
        output_tags = None
        enable_reuse = True
        if "arv:output_name" in job_order_object:
            output_name = job_order_object["arv:output_name"]
            del job_order_object["arv:output_name"]

        if "arv:output_tags" in job_order_object:
            output_tags = job_order_object["arv:output_tags"]
            del job_order_object["arv:output_tags"]

        if "arv:enable_reuse" in job_order_object:
            enable_reuse = job_order_object["arv:enable_reuse"]
            del job_order_object["arv:enable_reuse"]

        runner = arvados_cwl.ArvCwlRunner(api_client=arvados.api('v1', model=OrderedJsonModel()),
                                          output_name=output_name, output_tags=output_tags)

        t = load_tool(toolpath, runner.arv_make_tool)

        args = argparse.Namespace()
        args.project_uuid = arvados.current_job()["owner_uuid"]
        args.enable_reuse = enable_reuse
        args.submit = False
        args.debug = False
        args.quiet = False
        args.ignore_docker_for_reuse = False
        args.basedir = os.getcwd()
        args.name = None
        args.cwl_runner_job={"uuid": arvados.current_job()["uuid"], "state": arvados.current_job()["state"]}
        outputObj = runner.arv_executor(t, job_order_object, **vars(args))
    except Exception as e:
        if isinstance(e, WorkflowException):
            logging.info("Workflow error %s", e)
        else:
            logging.exception("Unhandled exception")
        if runner and runner.final_output_collection:
            outputCollection = runner.final_output_collection.portable_data_hash()
        else:
            outputCollection = None
        api.job_tasks().update(uuid=arvados.current_task()['uuid'],
                                             body={
                                                 'output': outputCollection,
                                                 'success': False,
                                                 'progress':1.0
                                             }).execute()
