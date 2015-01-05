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
    'crunch/node': 'User not found on host',
    'slurm/comm':  'Communication connection failure'
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
    if job['log']:
        log_collection = arvados.CollectionReader(job['log'], api)
        log_filename = "{}.log.txt".format(job['uuid'])
        return log_collection.open(log_filename).readlines()
    return []


user_names = {}
def job_user_name(api, user_uuid):
    def _lookup_user_name(api, user_uuid):
        try:
            return api.users().get(uuid=user_uuid).execute()['full_name']
        except arvados.errors.ApiError:
            return user_uuid

    if user_uuid not in user_names:
        user_names[user_uuid] = _lookup_user_name(api, user_uuid)
    return user_names[user_uuid]


job_pipeline_names = {}
def job_pipeline_name(api, job_uuid):
    def _lookup_pipeline_name(api, job_uuid):
        pipelines = api.pipeline_instances().list(
            filters='[["components", "like", "%{}%"]]'.format(job_uuid)).execute()
        if pipelines['items']:
            pi = pipelines['items'][0]
            if pi['name']:
                return pi['name']
            else:
                # Use the pipeline template name
                pt = api.pipeline_templates().get(uuid=pi['pipeline_template_uuid']).execute()
                if pt:
                    return pt['name']
        return ""

    if job_uuid not in job_pipeline_names:
        job_pipeline_names[job_uuid] = _lookup_pipeline_name(api, job_uuid)
    return job_pipeline_names[job_uuid]


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
        failure_type = None
        for i, lg in enumerate(logs):
            if is_failed_task(lg):
                # Get preceding log record to provide context.
                log_start = i - LOG_CONTEXT_LINES if i >= LOG_CONTEXT_LINES else 0
                log_end = i + 1
                lastlogs = ''.join(logs[log_start:log_end])
                # try to identify the type of failure.
                for key, rgx in JOB_FAILURE_TYPES.iteritems():
                    if re.search(rgx, lastlogs):
                        failure_type = key
                        break
            if failure_type is not None:
                break
        if failure_type is None:
            failure_type = 'unknown'
        failure_stats.setdefault(failure_type, set())
        failure_stats[failure_type].add(job_uuid)

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

    print "  {: <25s} {:4d}".format('Started',
                                    job_start_count)
    print "  {: <25s} {:4d} ({: >4.0%})".format('Successful',
                                                job_success_count,
                                                job_success_count / float(job_start_count))
    print "  {: <25s} {:4d} ({: >4.0%})".format('Failed',
                                                job_fail_count,
                                                job_fail_count / float(job_start_count))
    print "  {: <25s} {:4d} ({: >4.0%})".format('In progress',
                                                job_unfinished_count,
                                                job_unfinished_count / float(job_start_count))
    print ""

    # Report failure types.
    failure_summary = ""
    failure_detail = ""

    # Generate a mapping from failed job uuids to job records, to assist
    # in generating detailed statistics for job failures.
    jobs_failed_map = { job['uuid']: job for job in jobs_failed }

    # sort the failure stats in descending order by occurrence.
    sorted_failures = sorted(failure_stats.items(),
                             reverse=True,
                             key=lambda failed_job_list: len(failed_job_list))
    for failtype, job_uuids in sorted_failures:
        failstat = "  {: <25s} {:4d} ({: >4.0%})\n".format(
            failtype,
            len(job_uuids),
            len(job_uuids) / float(job_fail_count))
        failure_summary = failure_summary + failstat
        failure_detail = failure_detail + failstat
        for j in job_uuids:
            job_info = jobs_failed_map[j]
            job_owner = job_user_name(api, job_info['modified_by_user_uuid'])
            job_name = job_pipeline_name(api, job_info['uuid'])
            failure_detail = failure_detail + "    {}  {: <15.15s}  {:29.29s}\n".format(j, job_owner, job_name)
        failure_detail = failure_detail + "\n"

    print "Failures by class"
    print ""
    print failure_summary

    print "Failures by class (detail)"
    print ""
    print failure_detail

    return 0


if __name__ == "__main__":
    sys.exit(main())
