#!/usr/bin/env python

import sys
import logging
import argparse
import arvados
import json
from arvados.events import subscribe

def main(arguments=None):
    logger = logging.getLogger('arvados.arv-ws')

    parser = argparse.ArgumentParser()
    parser.add_argument('-u', '--uuid', type=str, default="")
    parser.add_argument('-f', '--filters', type=str, default="")

    group = parser.add_argument_group('group')
    group.add_argument('-p', '--pipeline', type=str, default="", help="Print log output from a pipeline and its jobs")
    group.add_argument('-j', '--job', type=str, default="", help="Print log output from a job")

    args = parser.parse_args(arguments)

    filters = []
    if args.uuid:
        filters += [ ['object_uuid', '=', args.uuid] ]

    if args.filters:
        filters += json.loads(args.filters)

    if args.pipeline:
        filters += [ ['object_uuid', '=', args.pipeline] ]

    if args.job:
        filters += [ ['object_uuid', '=', args.job], ['event_type', 'in', ['stderr', 'stdout'] ] ]

    api = arvados.api('v1', cache=False)

    known_component_jobs = set()
    def on_message(ev):
        print "\n"
        print ev
        if 'event_type' in ev and (args.pipeline or args.job):
            if ev['event_type'] in ('stderr', 'stdout'):
                print ev["properties"]["text"]
            elif ev["event_type"] in ("create", "update"):
                if args.job or ev["object_kind"] == "arvados#pipeline_instance":
                    if ev["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled"):
                        ws.close_connection()
                if ev["object_kind"] == "arvados#pipeline_instance":
                    pipeline_jobs = set()
                    for c in ev["properties"]["new_attributes"]["components"]:
                        if "job" in ev["properties"]["new_attributes"]["components"][c]:
                            pipeline_jobs.add(ev["properties"]["new_attributes"]["components"][c]["job"]["uuid"])
                    if known_component_jobs != pipeline_jobs:
                        ws.unsubscribe(filters)
                        filters = [['object_uuid', 'in', [args.pipeline] + list(pipeline_jobs)]]
                        ws.subscribe([['object_uuid', 'in', [args.pipeline] + list(pipeline_jobs)]])
                        known_component_jobs = pipeline_jobs
        else:
            print json.dumps(ev)

    ws = None
    try:
        print filters
        ws = subscribe(api, filters, lambda ev: on_message(ev))
        ws.run_forever()
    except KeyboardInterrupt:
        pass
    except Exception:
        logger.exception('')
    finally:
        if ws:
            ws.close_connection()
