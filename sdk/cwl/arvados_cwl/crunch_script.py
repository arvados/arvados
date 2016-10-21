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

logger = logging.getLogger('arvados.cwl-runner')

def run():
    # Print package versions
    logger.info(arvados_cwl.versionstring())

    api = arvados.api("v1")

    arvados_cwl.add_arv_hints()

    try:
        job_order_object = arvados.current_job()['script_parameters']

        pdh_path = re.compile(r'^[0-9a-f]{32}\+\d+(/.+)?$')

        def keeppath(v):
            if pdh_path.match(v):
                return "keep:%s" % v
            else:
                return v

        def keeppathObj(v):
            v["location"] = keeppath(v["location"])

        job_order_object["cwl:tool"] = "file://%s/%s" % (os.environ['TASK_KEEPMOUNT'], job_order_object["cwl:tool"])

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
        if "arv:output_name" in job_order_object:
            output_name = job_order_object["arv:output_name"]
            del job_order_object["arv:output_name"]

        runner = arvados_cwl.ArvCwlRunner(api_client=arvados.api('v1', model=OrderedJsonModel()),
                                          output_name=output_name)

        t = load_tool(job_order_object, runner.arv_make_tool)

        args = argparse.Namespace()
        args.project_uuid = arvados.current_job()["owner_uuid"]
        args.enable_reuse = True
        args.submit = False
        args.debug = True
        args.quiet = False
        args.ignore_docker_for_reuse = False
        args.basedir = os.getcwd()
        args.cwl_runner_job={"uuid": arvados.current_job()["uuid"], "state": arvados.current_job()["state"]}
        outputObj = runner.arv_executor(t, job_order_object, **vars(args))

        if runner.final_output_collection:
            outputCollection = runner.final_output_collection.portable_data_hash()
        else:
            outputCollection = None

        api.job_tasks().update(uuid=arvados.current_task()['uuid'],
                                             body={
                                                 'output': outputCollection,
                                                 'success': True,
                                                 'progress':1.0
                                             }).execute()
    except Exception as e:
        logging.exception("Unhandled exception")
        api.job_tasks().update(uuid=arvados.current_task()['uuid'],
                                             body={
                                                 'output': None,
                                                 'success': False,
                                                 'progress':1.0
                                             }).execute()
