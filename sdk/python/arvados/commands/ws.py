#!/usr/bin/env python

import sys
import logging
import argparse
import arvados
import json
from arvados.events import subscribe
import signal

def main(arguments=None):
    logger = logging.getLogger('arvados.arv-ws')

    parser = argparse.ArgumentParser()
    parser.add_argument('-u', '--uuid', type=str, default="", help="Filter events on object_uuid")
    parser.add_argument('-f', '--filters', type=str, default="", help="Arvados query filter to apply to log events (JSON encoded)")
    parser.add_argument('-s', '--start-time', type=str, default="", help="Arvados query filter to fetch log events created at or after this time. This will be server time in UTC. Allowed format: YYYY-MM-DD or YYYY-MM-DD hh:mm:ss")
    parser.add_argument('-i', '--id', type=int, default=None, help="Start from given log id.")

    group = parser.add_mutually_exclusive_group()
    group.add_argument('--poll-interval', default=15, type=int, help="If websockets is not available, specify the polling interval, default is every 15 seconds")
    group.add_argument('--no-poll', action='store_false', dest='poll_interval', help="Do not poll if websockets are not available, just fail")

    group = parser.add_mutually_exclusive_group()
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

    api = arvados.api('v1')

    if args.uuid:
        filters += [ ['object_uuid', '=', args.uuid] ]

    if args.filters:
        filters += json.loads(args.filters)

    if args.job:
        filters += [ ['object_uuid', '=', args.job] ]

    if args.pipeline:
        filters += [ ['object_uuid', '=', args.pipeline] ]

    if args.start_time:
        last_log_id = 1
        filters += [ ['created_at', '>=', args.start_time] ]
    else:
        last_log_id = None

    if args.id:
        last_log_id = args.id-1

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

                if ev["object_kind"] == "arvados#pipelineInstance" and args.pipeline:
                    if ev["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Paused"):
                        ws.close()

                if ev["object_kind"] == "arvados#job" and args.job:
                    if ev["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled"):
                        ws.close()
        elif 'status' in ev and ev['status'] == 200:
            pass
        else:
            print json.dumps(ev)

    try:
        ws = subscribe(arvados.api('v1'), filters, on_message, poll_fallback=args.poll_interval, last_log_id=last_log_id)
        if ws:
            if args.pipeline:
                c = api.pipeline_instances().get(uuid=args.pipeline).execute()
                update_subscribed_components(c["components"])
                if c["state"] in ("Complete", "Failed", "Paused"):
                    ws.close()
            ws.run_forever()
    except KeyboardInterrupt:
        pass
    except Exception as e:
        logger.error(e)
    finally:
        if ws:
            ws.close()
