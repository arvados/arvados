#! /usr/bin/env python

import argparse
import datetime
import json
import pprint
import sys

import arvados

def parse_arguments(arguments):
    arg_parser = argparse.ArgumentParser(
        description='Produce a report of Crunch failures within a specified time range')

    arg_parser.add_argument(
        '--start',
        help='Start date and time')
    arg_parser.add_argument(
        '--end',
        help='End date and time')
    arg_parser.add_argument(
        '--summary',
        action='append',
        default=[],
        help='SQL pattern (ILIKE syntax) to match on summary lines')

    return arg_parser.parse_args(arguments)

def api_timestamp(when=None):
    """Returns a string representing the timestamp 'when' in a format
    suitable for delivering to the API server.  Defaults to the
    current time.
    """
    if when is None:
        when = datetime.datetime.utcnow()
    return when.strftime("%Y-%m-%dT%H:%M:%SZ")

def main(arguments=None, stdout=sys.stdout, stderr=sys.stderr):
    args = parse_arguments(arguments)

    api = arvados.api('v1')

    now = datetime.datetime.utcnow()
    start_time = args.start or api_timestamp(now - datetime.timedelta(days=1))
    end_time = args.end or api_timestamp(now)

    summary_patterns = args.summary or ['%fail%']
    logfilters = [['summary', 'ilike', pattern] for pattern in summary_patterns ]
    logfilters.append( ['created_at', '>=', start_time] )
    logfilters.append( ['created_at', '<=', end_time] )

    logs = api.logs().list(
        filters=json.dumps(logfilters)
    ).execute()

    log_stats = {}
    for log in logs['items']:
        summary = log['summary']
        log_uuid = log['uuid']
        log_stats.setdefault(summary, []).append(log_uuid)

    # Sort the keys of log stats in decreasing order of frequency.
    for k in sorted(log_stats.keys(), cmp=lambda a,b: cmp(len(log_stats[b]), len(log_stats[a]))):
        print "{}: {}".format(k, len(log_stats[k]))


if __name__ == "__main__":
    sys.exit(main())
