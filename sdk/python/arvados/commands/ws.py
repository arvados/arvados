#!/usr/bin/env python

import sys
import logging
import argparse
import arvados
import json
import time
from arvados.events import subscribe

def main(arguments=None):
    logger = logging.getLogger('arvados.arv-ws')

    parser = argparse.ArgumentParser()
    parser.add_argument('-u', '--uuid', type=str, default="", help="Filter events on object_uuid")
    parser.add_argument('-f', '--filters', type=str, default="", help="Arvados query filter to apply to log events (JSON encoded)")

    group = parser.add_argument_group('Polling fallback')
    group.add_argument('--poll-interval', default=15, type=int, help="If websockets is not available, specify the polling interval, default is every 15 seconds")
    group.add_argument('--no-poll', action='store_false', dest='poll_interval', help="Do not poll if websockets are not available, just fail")

    group = parser.add_argument_group('Jobs and Pipelines')
    group.add_argument('-p', '--pipeline', type=str, default="", help="Supply pipeline uuid, print log output from pipeline and its jobs")
    group.add_argument('-j', '--job', type=str, default="", help="Supply job uuid, print log output from jobs")

    args = parser.parse_args(arguments)

    global filters
    filters = []
    if args.uuid:
        filters += [ ['object_uuid', '=', args.uuid] ]

    if args.filters:
        filters += json.loads(args.filters)

    if args.pipeline:
        filters += [ ['object_uuid', '=', args.pipeline] ]

    if args.job:
        filters += [ ['object_uuid', '=', args.job] ]

    api = arvados.api('v1', cache=False)

    global known_component_jobs
    global ws

    known_component_jobs = set()
    ws = None
    def on_message(ev):
        global known_component_jobs
        global filters
        global ws

        logger.debug(ev)
        if 'event_type' in ev and (args.pipeline or args.job):
            if ev['event_type'] in ('stderr', 'stdout'):
                sys.stdout.write(ev["properties"]["text"])
            elif ev["event_type"] in ("create", "update"):
                if ev["object_kind"] == "arvados#pipelineInstance":
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

    try:
        ws = subscribe(api, filters, on_message, poll_fallback=args.poll_interval)
        if ws:
            while True:
                time.sleep(60)
    except KeyboardInterrupt:
        pass
    except Exception:
        logger.exception('')
    finally:
        if ws:
            ws.close()
