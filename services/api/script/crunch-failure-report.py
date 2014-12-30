#! /usr/bin/env python

import argparse
import datetime
import json
import re
import sys

import arvados

# Useful configuration variables:

# The number of log lines preceding a job failure message that should
# be collected.
FAILED_JOB_LOG_LINES = 10

# Regex that signifies a failed job.
FAILED_JOB_REGEX = re.compile('fail')

# Regex that signifies a successful job.
SUCCESSFUL_JOB_REGEX = re.compile('finished')

# List of regexes by which to classify failures.
JOB_FAILURE_TYPES = [ 'User not found on host' ]

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
        '--match',
        default='fail',
        help='Regular expression to match on Crunch error output lines.')

    return arg_parser.parse_args(arguments)

def api_timestamp(when=None):
    """Returns a string representing the timestamp 'when' in a format
    suitable for delivering to the API server.  Defaults to the
    current time.
    """
    if when is None:
        when = datetime.datetime.utcnow()
    return when.strftime("%Y-%m-%dT%H:%M:%SZ")


def jobs_created_between_dates(api, start, end):
    return arvados.util.list_all(
        api.jobs().list,
        filters=json.dumps([ ['created_at', '>=', start],
                             ['created_at', '<=', end] ]))


def job_error_logs(api, job_uuid):
    return arvados.util.list_all(
        api.logs().list,
        filters=json.dumps([ ['object_uuid', '=', job_uuid],
                             ['event_type', '=', 'stderr'] ]))


def is_failed_job(logline):
    return FAILED_JOB_REGEX.search(logline) != None


def is_successful_job(logline):
    return SUCCESSFUL_JOB_REGEX.search(logline) != None

def log(s):
    print >>sys.stderr, "{}: {}".format(datetime.datetime.utcnow(), s)

def main(arguments=None, stdout=sys.stdout, stderr=sys.stderr):
    args = parse_arguments(arguments)

    api = arvados.api('v1')

    now = datetime.datetime.utcnow()
    start_time = args.start or api_timestamp(now - datetime.timedelta(days=1))
    end_time = args.end or api_timestamp(now)

    # Find all jobs created within the specified window,
    # and their corresponding job logs.
    log("fetching jobs between {} and {}".format(start_time, end_time))
    jobs_created = jobs_created_between_dates(api, start_time, end_time)
    log("jobs created: {}".format(len(jobs_created)))


    # Find failed jobs and record the job failure text.
    jobs_successful = set()
    jobs_failed = set()
    jobs_failed_types = {}
    for j in jobs_created:
        # Skip this log entry if we've already recorded
        # the job failure.
        job_uuid = j['uuid']
        if job_uuid in jobs_failed:
            continue
        logs = job_error_logs(api, job_uuid)
        log("fetched {} job error logs for {}".format(len(logs), job_uuid))
        # If this line marks a failed job, record it and
        # the preceding log lines.
        for i, lg in enumerate(logs):
            if is_failed_job(lg['properties']['text']):
                jobs_failed.add(job_uuid)
                # Classify this job failure.
                lastlogs = "\n".join(
                    [ l['properties']['text'] for l in logs[i-FAILED_JOB_LOG_LINES:i] ])
                log("searching job {} lastlogs: {}".format(job_uuid, lastlogs))
                for failtype in JOB_FAILURE_TYPES:
                    if re.search(failtype, lastlogs):
                        jobs_failed_types.setdefault(failtype, set())
                        jobs_failed_types[failtype].add(job_uuid)
                        continue
                    # no specific reason found
                    jobs_failed_types.setdefault('unknown', set())
                    jobs_failed_types['unknown'].add(job_uuid)
                break
            elif is_successful_job(lg['properties']['text']):
                jobs_successful.add(job_uuid)
                break

    # Report percentages of successful, failed and unfinished jobs.
    job_start_count = len(jobs_created)
    job_success_count = len(jobs_successful)
    job_fail_count = len(jobs_failed)
    job_unfinished_count = job_start_count - job_success_count - job_fail_count

    print "Started:     {0:4d}".format(job_start_count)
    print "Successful:  {0:4d} ({1:3.0%})".format(job_success_count, job_success_count / float(job_start_count))
    print "Failed:      {0:4d} ({1:3.0%})".format(job_fail_count, job_fail_count / float(job_start_count))
    print "In progress: {0:4d} ({1:3.0%})".format(job_unfinished_count, job_unfinished_count / float(job_start_count))

    # Report failure types.
    for failtype in jobs_failed_types:
        print "{0:20s}: {1:4d} ({2:3.0%})".format(
            failtype, len(jobs_failed_types), len(jobs_failed_types) / float(job_fail_count))

if __name__ == "__main__":
    sys.exit(main())
