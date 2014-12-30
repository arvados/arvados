#! /usr/bin/env python

import argparse
import datetime
import json
import re
import sys

import arvados

# Useful configuration variables:

# Number of log lines to use as context in diagnosing failure.
LOG_CONTEXT_LINES = 10

# Regex that signifies a failed task.
FAILED_TASK_REGEX = re.compile(' \d+ failure (.*permanent)')

# Regular expressions used to classify failure types.
JOB_FAILURE_TYPES = {
    'sys/docker': 'Cannot destroy container',
    'crunch/node': 'User not found on host'
}

def parse_arguments(arguments):
    arg_parser = argparse.ArgumentParser(
        description='Produce a report of Crunch failures within a specified time range')

    arg_parser.add_argument(
        '--start',
        help='Start date and time')
    arg_parser.add_argument(
        '--end',
        help='End date and time')

    args = arg_parser.parse_args(arguments)

    if args.start and not is_valid_timestamp(args.start):
        raise ValueError(args.start)
    if args.end and not is_valid_timestamp(args.end):
        raise ValueError(args.end)

    return args


def api_timestamp(when=None):
    """Returns a string representing the timestamp 'when' in a format
    suitable for delivering to the API server.  Defaults to the
    current time.
    """
    if when is None:
        when = datetime.datetime.utcnow()
    return when.strftime("%Y-%m-%dT%H:%M:%SZ")


def is_valid_timestamp(ts):
    return re.match(r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z', ts)

def jobs_created_between_dates(api, start, end):
    return arvados.util.list_all(
        api.jobs().list,
        filters=json.dumps([ ['created_at', '>=', start],
                             ['created_at', '<=', end] ]))


def job_logs(api, job):
    # Returns the contents of the log for this job (as an array of lines).
    log_filename = "{}.log.txt".format(job['uuid'])
    log_collection = arvados.CollectionReader(job['log'], api)
    return log_collection.open(log_filename).readlines()


def is_failed_task(logline):
    return FAILED_TASK_REGEX.search(logline) != None


def main(arguments=None, stdout=sys.stdout, stderr=sys.stderr):
    args = parse_arguments(arguments)

    api = arvados.api('v1')

    now = datetime.datetime.utcnow()
    start_time = args.start or api_timestamp(now - datetime.timedelta(days=1))
    end_time = args.end or api_timestamp(now)

    # Find all jobs created within the specified window,
    # and their corresponding job logs.
    jobs_created = jobs_created_between_dates(api, start_time, end_time)
    jobs_failed     = [job for job in jobs_created if job['state'] == 'Failed']
    jobs_successful = [job for job in jobs_created if job['state'] == 'Complete']

    # Find failed jobs and record the job failure text.

    # failure_stats maps failure types (e.g. "sys/docker") to
    # a set of job UUIDs that failed for that reason.
    failure_stats = {}
    for job in jobs_failed:
        job_uuid = job['uuid']
        logs = job_logs(api, job)
        # Find the first permanent task failure, and collect the
        # preceding log lines.
        for i, lg in enumerate(logs):
            if is_failed_task(lg):
                # Get preceding log record to provide context.
                log_start = i - LOG_CONTEXT_LINES if i >= LOG_CONTEXT_LINES else 0
                log_end = i + 1
                lastlogs = ''.join(logs[log_start:log_end])
                # try to identify the type of failure.
                failure_type = 'unknown'
                for key, rgx in JOB_FAILURE_TYPES.iteritems():
                    if re.search(rgx, lastlogs):
                        failure_type = key
                        break
                failure_stats.setdefault(failure_type, set())
                failure_stats[failure_type].add(job_uuid)
                break
            # If we got here, the job is recorded as "failed" but we
            # could not find the failure of any specific task.
            failure_stats.setdefault('unknown', set())
            failure_stats['unknown'].add(job_uuid)

    # Report percentages of successful, failed and unfinished jobs.
    print "Start: {:20s}".format(start_time)
    print "End:   {:20s}".format(end_time)
    print ""

    print "Overview"
    print ""

    job_start_count = len(jobs_created)
    job_success_count = len(jobs_successful)
    job_fail_count = len(jobs_failed)
    job_unfinished_count = job_start_count - job_success_count - job_fail_count

    print "  Started:     {:4d}".format(job_start_count)
    print "  Successful:  {:4d} ({:3.0%})".format(
        job_success_count, job_success_count / float(job_start_count))
    print "  Failed:      {:4d} ({:3.0%})".format(
        job_fail_count, job_fail_count / float(job_start_count))
    print "  In progress: {:4d} ({:3.0%})".format(
        job_unfinished_count, job_unfinished_count / float(job_start_count))
    print ""

    # Report failure types.
    failure_summary = ""
    failure_detail = ""

    for failtype, job_uuids in failure_stats.iteritems():
        failstat = "  {:s} {:4d} ({:3.0%})\n".format(
            failtype, len(job_uuids), len(job_uuids) / float(job_fail_count))
        failure_summary = failure_summary + failstat
        failure_detail = failure_detail + failstat
        for j in job_uuids:
            failure_detail = failure_detail + "    http://crvr.se/{}\n".format(j)
        failure_detail = failure_detail + "\n"

    print "Failures by class"
    print ""
    print failure_summary

    print "Failures by class (detail):"
    print ""
    print failure_detail

    return 0


if __name__ == "__main__":
    sys.exit(main())
