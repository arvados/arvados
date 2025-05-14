# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# arv-copy [--recursive] [--no-recursive] object-uuid
#
# Copies an object from Arvados instance src to instance dst.
#
# By default, arv-copy recursively copies any dependent objects
# necessary to make the object functional in the new instance
# (e.g. for a workflow, arv-copy copies the workflow,
# input collections, and docker images). If
# --no-recursive is given, arv-copy copies only the single record
# identified by object-uuid.
#
# The user must have configuration files {src}.conf and
# {dst}.conf in a standard configuration directory with valid login credentials
# for instances src and dst.  If either of these files is not found,
# arv-copy will issue an error.

import argparse
import os
import re
import subprocess
import sys
import logging

import arvados
import arvados.config
import arvados.keep
import arvados.util
import arvados.commands._util as arv_cmd
import arvados.commands.keepdocker
from arvados.logging import log_handler

from arvados._internal import http_to_keep, s3_to_keep, to_keep_util
from arvados._internal.arvcopy import api_for_instance, copy_workflow, copy_collection, copy_project

from arvados._version import __version__

COMMIT_HASH_RE = re.compile(r'^[0-9a-f]{1,40}$')

arvlogger = logging.getLogger('arvados')
keeplogger = logging.getLogger('arvados.keep')
logger = logging.getLogger('arvados.arv-copy')

# Set this up so connection errors get logged.
googleapi_logger = logging.getLogger('googleapiclient.http')

# Set of (repository, script_version) two-tuples of commits copied in git.
scripts_copied = set()

# The owner_uuid of the object being copied
src_owner_uuid = None

def main():
    copy_opts = argparse.ArgumentParser(add_help=False)

    copy_opts.add_argument(
        '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
        help='Print version and exit.')
    copy_opts.add_argument(
        '-v', '--verbose', dest='verbose', action='store_true',
        help='Verbose output.')
    copy_opts.add_argument(
        '--progress', dest='progress', action='store_true',
        help='Report progress on copying collections. (default)')
    copy_opts.add_argument(
        '--no-progress', dest='progress', action='store_false',
        help='Do not report progress on copying collections.')
    copy_opts.add_argument(
        '-f', '--force', dest='force', action='store_true',
        help='Perform copy even if the object appears to exist at the remote destination.')
    copy_opts.add_argument(
        '--src', dest='source_arvados',
        help="""
Client configuration location for the source Arvados cluster.
May be either a configuration file path, or a plain identifier like `foo`
to search for a configuration file `foo.conf` under a systemd or XDG configuration directory.
If not provided, will search for a configuration file named after the cluster ID of the source object UUID.
""",
    )
    copy_opts.add_argument(
        '--dst', dest='destination_arvados',
        help="""
Client configuration location for the destination Arvados cluster.
May be either a configuration file path, or a plain identifier like `foo`
to search for a configuration file `foo.conf` under a systemd or XDG configuration directory.
If not provided, will use the default client configuration from the environment or `settings.conf`.
""",
    )
    copy_opts.add_argument(
        '--recursive', dest='recursive', action='store_true',
        help='Recursively copy any dependencies for this object, and subprojects. (default)')
    copy_opts.add_argument(
        '--no-recursive', dest='recursive', action='store_false',
        help='Do not copy any dependencies or subprojects.')
    copy_opts.add_argument(
        '--project-uuid', dest='project_uuid',
        help='The UUID of the project at the destination to which the collection or workflow should be copied.')
    copy_opts.add_argument(
        '--replication',
        type=arv_cmd.RangedValue(int, range(1, sys.maxsize)),
        metavar='N',
        help="""
Number of replicas per storage class for the copied collections at the destination.
If not provided (or if provided with invalid value),
use the destination's default replication-level setting (if found),
or the fallback value 2.
""")
    copy_opts.add_argument(
        '--storage-classes',
        type=arv_cmd.UniqueSplit(),
        help='Comma separated list of storage classes to be used when saving data to the destinaton Arvados instance.')
    copy_opts.add_argument("--varying-url-params", type=str, default="",
                        help="A comma separated list of URL query parameters that should be ignored when storing HTTP URLs in Keep.")

    copy_opts.add_argument("--prefer-cached-downloads", action="store_true", default=False,
                        help="If a HTTP URL is found in Keep, skip upstream URL freshness check (will not notice if the upstream has changed, but also not error if upstream is unavailable).")

    copy_opts.add_argument(
        'object_uuid',
        help='The UUID of the object to be copied.')
    copy_opts.set_defaults(progress=True)
    copy_opts.set_defaults(recursive=True)

    parser = argparse.ArgumentParser(
        description='Copy a workflow, collection or project from one Arvados instance to another.  On success, the uuid of the copied object is printed to stdout.',
        parents=[copy_opts, arv_cmd.retry_opt])
    args = parser.parse_args()

    args.export_all_fields = False

    if args.verbose:
        arvlogger.setLevel(logging.DEBUG)
    else:
        arvlogger.setLevel(logging.INFO)
        keeplogger.setLevel(logging.WARNING)

    if not args.source_arvados and arvados.util.uuid_pattern.match(args.object_uuid):
        args.source_arvados = args.object_uuid[:5]

    if not args.destination_arvados and args.project_uuid:
        args.destination_arvados = args.project_uuid[:5]


    # Create API clients for the source and destination instances
    src_arv = api_for_instance(args.source_arvados, args.retries)
    dst_arv = api_for_instance(args.destination_arvados, args.retries)

    # Once we've successfully contacted the clusters, we probably
    # don't want to see logging about retries (unless the user asked
    # for verbose output).
    if not args.verbose:
        googleapi_logger.setLevel(logging.ERROR)

    if src_arv.config()["ClusterID"] == dst_arv.config()["ClusterID"]:
        logger.info("Copying within cluster %s", src_arv.config()["ClusterID"])
    else:
        logger.info("Source cluster is %s", src_arv.config()["ClusterID"])
        logger.info("Destination cluster is %s", dst_arv.config()["ClusterID"])

    if not args.project_uuid:
        args.project_uuid = dst_arv.users().current().execute(num_retries=args.retries)["uuid"]

    # Identify the kind of object we have been given, and begin copying.
    t = uuid_type(src_arv, args.object_uuid)

    try:
        if t == 'Collection':
            set_src_owner_uuid(src_arv.collections(), args.object_uuid, args)
            result = copy_collection(args.object_uuid,
                                     src_arv, dst_arv,
                                     args)
        elif t == 'Workflow':
            set_src_owner_uuid(src_arv.workflows(), args.object_uuid, args)
            result = copy_workflow(args.object_uuid, src_arv, dst_arv, args)
        elif t == 'Group':
            set_src_owner_uuid(src_arv.groups(), args.object_uuid, args)
            result = copy_project(args.object_uuid, src_arv, dst_arv, args.project_uuid, args)
        elif t == 'httpURL' or t == 's3URL':
            result = copy_from_url(args.object_uuid, src_arv, dst_arv, args)
        else:
            abort("cannot copy object {} of type {}".format(args.object_uuid, t))
    except Exception as e:
        logger.error("%s", e, exc_info=args.verbose)
        exit(1)

    if not result:
        exit(1)

    # If no exception was thrown and the response does not have an
    # error_token field, presume success
    if result is None or 'error_token' in result or 'uuid' not in result:
        if result:
            logger.error("API server returned an error result: {}".format(result))
        exit(1)

    print(result['uuid'])

    if result.get('partial_error'):
        logger.warning("Warning: created copy with uuid {} but failed to copy some items: {}".format(result['uuid'], result['partial_error']))
        exit(1)

    logger.info("Success: created copy with uuid {}".format(result['uuid']))
    exit(0)

def set_src_owner_uuid(resource, uuid, args):
    global src_owner_uuid
    c = resource.get(uuid=uuid).execute(num_retries=args.retries)
    src_owner_uuid = c.get("owner_uuid")


# git_rev_parse(rev, repo)
#
#    Returns the 40-character commit hash corresponding to 'rev' in
#    git repository 'repo' (which must be the path of a local git
#    repository)
#
def git_rev_parse(rev, repo):
    proc = subprocess.run(
        ['git', 'rev-parse', rev],
        check=True,
        cwd=repo,
        stdout=subprocess.PIPE,
        text=True,
    )
    return proc.stdout.read().strip()

# uuid_type(api, object_uuid)
#
#    Returns the name of the class that object_uuid belongs to, based on
#    the second field of the uuid.  This function consults the api's
#    schema to identify the object class.
#
#    It returns a string such as 'Collection', 'Workflow', etc.
#
#    Special case: if handed a Keep locator hash, return 'Collection'.
#
def uuid_type(api, object_uuid):
    if re.match(arvados.util.keep_locator_pattern, object_uuid):
        return 'Collection'

    if object_uuid.startswith("http:") or object_uuid.startswith("https:"):
        return 'httpURL'

    if object_uuid.startswith("s3:"):
        return 's3URL'

    p = object_uuid.split('-')
    if len(p) == 3:
        type_prefix = p[1]
        for k in api._schema.schemas:
            obj_class = api._schema.schemas[k].get('uuidPrefix', None)
            if type_prefix == obj_class:
                return k
    return None


def copy_from_url(url, src, dst, args):

    project_uuid = args.project_uuid
    # Ensure string of varying parameters is well-formed
    prefer_cached_downloads = args.prefer_cached_downloads

    cached = to_keep_util.CheckCacheResult(None, None, None, None, None)

    if url.startswith("http:") or url.startswith("https:"):
        cached = http_to_keep.check_cached_url(src, project_uuid, url, {},
                                               varying_url_params=args.varying_url_params,
                                               prefer_cached_downloads=prefer_cached_downloads)
    elif url.startswith("s3:"):
        import boto3.session
        botosession = boto3.session.Session()
        cached = s3_to_keep.check_cached_url(src, botosession, project_uuid, url, {},
                                             prefer_cached_downloads=prefer_cached_downloads)

    if cached[2] is not None:
        return copy_collection(cached[2], src, dst, args)

    if url.startswith("http:") or url.startswith("https:"):
        cached = http_to_keep.http_to_keep(dst, project_uuid, url,
                                           varying_url_params=args.varying_url_params,
                                           prefer_cached_downloads=prefer_cached_downloads)
    elif url.startswith("s3:"):
        cached = s3_to_keep.s3_to_keep(dst, botosession, project_uuid, url,
                                       prefer_cached_downloads=prefer_cached_downloads)

    if cached is not None:
        return {"uuid": cached[2]}


def abort(msg, code=1):
    logger.info("arv-copy: %s", msg)
    exit(code)


# Code for reporting on the progress of a collection upload.
# Stolen from arvados.commands.put.ArvPutCollectionWriter
# TODO(twp): figure out how to refactor into a shared library
# (may involve refactoring some arvados.commands.arv_copy.copy_collection
# code)

def machine_progress(obj_uuid, bytes_written, bytes_expected):
    return "{} {}: {} {} written {} total\n".format(
        sys.argv[0],
        os.getpid(),
        obj_uuid,
        bytes_written,
        -1 if (bytes_expected is None) else bytes_expected)

def human_progress(obj_uuid, bytes_written, bytes_expected):
    if bytes_expected:
        return "\r{}: {}M / {}M {:.1%} ".format(
            obj_uuid,
            bytes_written >> 20, bytes_expected >> 20,
            float(bytes_written) / bytes_expected)
    else:
        return "\r{}: {} ".format(obj_uuid, bytes_written)

class ProgressWriter(object):
    _progress_func = None
    outfile = sys.stderr

    def __init__(self, progress_func):
        self._progress_func = progress_func

    def report(self, obj_uuid, bytes_written, bytes_expected):
        if self._progress_func is not None:
            self.outfile.write(
                self._progress_func(obj_uuid, bytes_written, bytes_expected))

    def finish(self):
        self.outfile.write("\n")

if __name__ == '__main__':
    main()
