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
    global known_component_jobs
    global ws

    filters = []
    known_component_jobs = set()
    ws = None

    def update_subscribed_components(components):
        global known_component_jobs
        global filters
        pipeline_jobs = set()
        for c in components:
            if "job" in components[c]:
                pipeline_jobs.add(components[c]["job"]["uuid"])
        if known_component_jobs != pipeline_jobs:
            ws.unsubscribe(filters)
            filters = [['object_uuid', 'in', [args.pipeline] + list(pipeline_jobs)]]
            ws.subscribe([['object_uuid', 'in', [args.pipeline] + list(pipeline_jobs)]])
            known_component_jobs = pipeline_jobs

    api = arvados.api('v1', cache=False)

    if args.uuid:
        filters += [ ['object_uuid', '=', args.uuid] ]

    if args.filters:
        filters += json.loads(args.filters)

    if args.job:
        filters += [ ['object_uuid', '=', args.job] ]

    if args.pipeline:
        filters += [ ['object_uuid', '=', args.pipeline] ]

    def on_message(ev):
        global filters
        global ws

        logger.debug(ev)
        if 'event_type' in ev and (args.pipeline or args.job):
            if ev['event_type'] in ('stderr', 'stdout'):
                sys.stdout.write(ev["properties"]["text"])
            elif ev["event_type"] in ("create", "update"):
                if ev["object_kind"] == "arvados#pipelineInstance":
                    update_subscribed_components(ev["properties"]["new_attributes"]["components"])
        elif 'status' in ev and ev['status'] == 200:
            pass
        else:
            print json.dumps(ev)

    try:
        ws = subscribe(api, filters, on_message, poll_fallback=args.poll_interval)
        if ws:
            if args.pipeline:
                c = api.pipeline_instances().get(uuid=args.pipeline).execute()
                update_subscribed_components(c["components"])

            while True:
                time.sleep(60)
    except KeyboardInterrupt:
        pass
    except Exception as e:
        logger.error(e)
    finally:
        if ws:
            ws.close()
