#! /usr/bin/env python

# arv-copy [--recursive] [--no-recursive] object-uuid src dst
#
# Copies an object from Arvados instance src to instance dst.
#
# By default, arv-copy recursively copies any dependent objects
# necessary to make the object functional in the new instance
# (e.g. for a pipeline instance, arv-copy copies the pipeline
# template, input collection, docker images, git repositories). If
# --no-recursive is given, arv-copy copies only the single record
# identified by object-uuid.
#
# The user must have files $HOME/.config/arvados/{src}.conf and
# $HOME/.config/arvados/{dst}.conf with valid login credentials for
# instances src and dst.  If either of these files is not found,
# arv-copy will issue an error.

import argparse
import getpass
import os
import re
import shutil
import sys
import logging
import tempfile

import arvados
import arvados.config
import arvados.keep
import arvados.util

logger = logging.getLogger('arvados.arv-copy')

# local_repo_dir records which git repositories from the Arvados source
# instance have been checked out locally during this run, and to which
# directories.
# e.g. if repository 'twp' from src_arv has been cloned into
# /tmp/gitfHkV9lu44A then local_repo_dir['twp'] = '/tmp/gitfHkV9lu44A'
#
local_repo_dir = {}

# List of collections that have been copied in this session, and their
# destination collection UUIDs.
collections_copied = {}

def main():
    parser = argparse.ArgumentParser(
        description='Copy a pipeline instance, template or collection from one Arvados instance to another.')

    parser.add_argument(
        '-v', '--verbose', dest='verbose', action='store_true',
        help='Verbose output.')
    parser.add_argument(
        '--progress', dest='progress', action='store_true',
        help='Report progress on copying collections. (default)')
    parser.add_argument(
        '--no-progress', dest='progress', action='store_false',
        help='Do not report progress on copying collections.')
    parser.add_argument(
        '-f', '--force', dest='force', action='store_true',
        help='Perform copy even if the object appears to exist at the remote destination.')
    parser.add_argument(
        '--src', dest='source_arvados', required=True,
        help='The name of the source Arvados instance (required). May be either a pathname to a config file, or the basename of a file in $HOME/.config/arvados/instance_name.conf.')
    parser.add_argument(
        '--dst', dest='destination_arvados', required=True,
        help='The name of the destination Arvados instance (required). May be either a pathname to a config file, or the basename of a file in $HOME/.config/arvados/instance_name.conf.')
    parser.add_argument(
        '--recursive', dest='recursive', action='store_true',
        help='Recursively copy any dependencies for this object. (default)')
    parser.add_argument(
        '--no-recursive', dest='recursive', action='store_false',
        help='Do not copy any dependencies. NOTE: if this option is given, the copied object will need to be updated manually in order to be functional.')
    parser.add_argument(
        '--dst-git-repo', dest='dst_git_repo',
        help='The name of the destination git repository. Required when copying a pipeline recursively.')
    parser.add_argument(
        '--project-uuid', dest='project_uuid',
        help='The UUID of the project at the destination to which the pipeline should be copied.')
    parser.add_argument(
        'object_uuid',
        help='The UUID of the object to be copied.')
    parser.set_defaults(progress=True)
    parser.set_defaults(recursive=True)

    args = parser.parse_args()

    if args.verbose:
        logger.setLevel(logging.DEBUG)
    else:
        logger.setLevel(logging.INFO)

    # Create API clients for the source and destination instances
    src_arv = api_for_instance(args.source_arvados)
    dst_arv = api_for_instance(args.destination_arvados)

    # Identify the kind of object we have been given, and begin copying.
    t = uuid_type(src_arv, args.object_uuid)
    if t == 'Collection':
        result = copy_collection(args.object_uuid,
                                 src_arv, dst_arv,
                                 args)
    elif t == 'PipelineInstance':
        result = copy_pipeline_instance(args.object_uuid,
                                        src_arv, dst_arv,
                                        args)
    elif t == 'PipelineTemplate':
        result = copy_pipeline_template(args.object_uuid,
                                        src_arv, dst_arv, args)
    else:
        abort("cannot copy object {} of type {}".format(args.object_uuid, t))

    # Clean up any outstanding temp git repositories.
    for d in local_repo_dir.values():
        shutil.rmtree(d, ignore_errors=True)

    # If no exception was thrown and the response does not have an
    # error_token field, presume success
    if 'error_token' in result or 'uuid' not in result:
        logger.error("API server returned an error result: {}".format(result))
        exit(1)

    logger.info("")
    logger.info("Success: created copy with uuid {}".format(result['uuid']))
    exit(0)

# api_for_instance(instance_name)
#
#     Creates an API client for the Arvados instance identified by
#     instance_name.
#
#     If instance_name contains a slash, it is presumed to be a path
#     (either local or absolute) to a file with Arvados configuration
#     settings.
#
#     Otherwise, it is presumed to be the name of a file in
#     $HOME/.config/arvados/instance_name.conf
#
def api_for_instance(instance_name):
    if '/' in instance_name:
        config_file = instance_name
    else:
        config_file = os.path.join(os.environ['HOME'], '.config', 'arvados', "{}.conf".format(instance_name))

    try:
        cfg = arvados.config.load(config_file)
    except (IOError, OSError) as e:
        abort(("Could not open config file {}: {}\n" +
               "You must make sure that your configuration tokens\n" +
               "for Arvados instance {} are in {} and that this\n" +
               "file is readable.").format(
                   config_file, e, instance_name, config_file))

    if 'ARVADOS_API_HOST' in cfg and 'ARVADOS_API_TOKEN' in cfg:
        api_is_insecure = (
            cfg.get('ARVADOS_API_HOST_INSECURE', '').lower() in set(
                ['1', 't', 'true', 'y', 'yes']))
        client = arvados.api('v1',
                             host=cfg['ARVADOS_API_HOST'],
                             token=cfg['ARVADOS_API_TOKEN'],
                             insecure=api_is_insecure,
                             cache=False)
    else:
        abort('need ARVADOS_API_HOST and ARVADOS_API_TOKEN for {}'.format(instance_name))
    return client

# copy_pipeline_instance(pi_uuid, src, dst, args)
#
#    Copies a pipeline instance identified by pi_uuid from src to dst.
#
#    If the args.recursive option is set:
#      1. Copies all input collections
#           * For each component in the pipeline, include all collections
#             listed as job dependencies for that component)
#      2. Copy docker images
#      3. Copy git repositories
#      4. Copy the pipeline template
#
#    The only changes made to the copied pipeline instance are:
#      1. The original pipeline instance UUID is preserved in
#         the 'properties' hash as 'copied_from_pipeline_instance_uuid'.
#      2. The pipeline_template_uuid is changed to the new template uuid.
#      3. The owner_uuid of the instance is changed to the user who
#         copied it.
#
def copy_pipeline_instance(pi_uuid, src, dst, args):
    # Fetch the pipeline instance record.
    pi = src.pipeline_instances().get(uuid=pi_uuid).execute()

    if args.recursive:
        if not args.dst_git_repo:
            abort('--dst-git-repo is required when copying a pipeline recursively.')
        # Copy the pipeline template and save the copied template.
        if pi.get('pipeline_template_uuid', None):
            pt = copy_pipeline_template(pi['pipeline_template_uuid'],
                                        src, dst, args)

        # Copy input collections, docker images and git repos.
        pi = copy_collections(pi, src, dst, args)
        copy_git_repos(pi, src, dst, args.dst_git_repo)

        # Update the fields of the pipeline instance with the copied
        # pipeline template.
        if pi.get('pipeline_template_uuid', None):
            pi['pipeline_template_uuid'] = pt['uuid']

    else:
        # not recursive
        logger.info("Copying only pipeline instance %s.", pi_uuid)
        logger.info("You are responsible for making sure all pipeline dependencies have been updated.")

    # Update the pipeline instance properties, and create the new
    # instance at dst.
    pi['properties']['copied_from_pipeline_instance_uuid'] = pi_uuid
    pi['description'] = "Pipeline copied from {}\n\n{}".format(
        pi_uuid, pi.get('description', ''))
    if args.project_uuid:
        pi['owner_uuid'] = args.project_uuid
    else:
        del pi['owner_uuid']
    del pi['uuid']

    new_pi = dst.pipeline_instances().create(body=pi, ensure_unique_name=True).execute()
    return new_pi

# copy_pipeline_template(pt_uuid, src, dst, args)
#
#    Copies a pipeline template identified by pt_uuid from src to dst.
#
#    If args.recursive is True, also copy any collections, docker
#    images and git repositories that this template references.
#
#    The owner_uuid of the new template is changed to that of the user
#    who copied the template.
#
#    Returns the copied pipeline template object.
#
def copy_pipeline_template(pt_uuid, src, dst, args):
    # fetch the pipeline template from the source instance
    pt = src.pipeline_templates().get(uuid=pt_uuid).execute()

    if args.recursive:
        if not args.dst_git_repo:
            abort('--dst-git-repo is required when copying a pipeline recursively.')
        # Copy input collections, docker images and git repos.
        pt = copy_collections(pt, src, dst, args)
        copy_git_repos(pt, src, dst, args.dst_git_repo)

    pt['description'] = "Pipeline template copied from {}\n\n{}".format(
        pt_uuid, pt.get('description', ''))
    pt['name'] = "{} copied from {}".format(pt.get('name', ''), pt_uuid)
    del pt['uuid']
    del pt['owner_uuid']

    return dst.pipeline_templates().create(body=pt, ensure_unique_name=True).execute()

# copy_collections(obj, src, dst, args)
#
#    Recursively copies all collections referenced by 'obj' from src
#    to dst.  obj may be a dict or a list, in which case we run
#    copy_collections on every value it contains. If it is a string,
#    search it for any substring that matches a collection hash or uuid
#    (this will find hidden references to collections like
#      "input0": "$(file 3229739b505d2b878b62aed09895a55a+142/HWI-ST1027_129_D0THKACXX.1_1.fastq)")
#
#    Returns a copy of obj with any old collection uuids replaced by
#    the new ones.
#
def copy_collections(obj, src, dst, args):

    def copy_collection_fn(collection_match):
        """Helper function for regex substitution: copies a single collection,
        identified by the collection_match MatchObject, to the
        destination.  Returns the destination collection uuid (or the
        portable data hash if that's what src_id is).

        """
        src_id = collection_match.group(0)
        if src_id not in collections_copied:
            dst_col = copy_collection(src_id, src, dst, args)
            if src_id in [dst_col['uuid'], dst_col['portable_data_hash']]:
                collections_copied[src_id] = src_id
            else:
                collections_copied[src_id] = dst_col['uuid']
        return collections_copied[src_id]

    if isinstance(obj, basestring):
        # Copy any collections identified in this string to dst, replacing
        # them with the dst uuids as necessary.
        obj = arvados.util.portable_data_hash_pattern.sub(copy_collection_fn, obj)
        obj = arvados.util.collection_uuid_pattern.sub(copy_collection_fn, obj)
        return obj
    elif type(obj) == dict:
        return {v: copy_collections(obj[v], src, dst, args) for v in obj}
    elif type(obj) == list:
        return [copy_collections(v, src, dst, args) for v in obj]
    return obj

# copy_git_repos(p, src, dst, dst_repo)
#
#    Copies all git repositories referenced by pipeline instance or
#    template 'p' from src to dst.
#
#    For each component c in the pipeline:
#      * Copy git repositories named in c['repository'] and c['job']['repository'] if present
#      * Rename script versions:
#          * c['script_version']
#          * c['job']['script_version']
#          * c['job']['supplied_script_version']
#        to the commit hashes they resolve to, since any symbolic
#        names (tags, branches) are not preserved in the destination repo.
#
#    The pipeline object is updated in place with the new repository
#    names.  The return value is undefined.
#
def copy_git_repos(p, src, dst, dst_repo):
    copied = set()
    for c in p['components']:
        component = p['components'][c]
        if 'repository' in component:
            repo = component['repository']
            script_version = component.get('script_version', None)
            if repo not in copied:
                copy_git_repo(repo, src, dst, dst_repo, script_version)
                copied.add(repo)
            component['repository'] = dst_repo
            if script_version:
                repo_dir = local_repo_dir[repo]
                component['script_version'] = git_rev_parse(script_version, repo_dir)
        if 'job' in component:
            j = component['job']
            if 'repository' in j:
                repo = j['repository']
                script_version = j.get('script_version', None)
                if repo not in copied:
                    copy_git_repo(repo, src, dst, dst_repo, script_version)
                    copied.add(repo)
                j['repository'] = dst_repo
                repo_dir = local_repo_dir[repo]
                if script_version:
                    j['script_version'] = git_rev_parse(script_version, repo_dir)
                if 'supplied_script_version' in j:
                    j['supplied_script_version'] = git_rev_parse(j['supplied_script_version'], repo_dir)

def total_collection_size(manifest_text):
    """Return the total number of bytes in this collection (excluding
    duplicate blocks)."""

    total_bytes = 0
    locators_seen = {}
    for line in manifest_text.splitlines():
        words = line.split()
        for word in words[1:]:
            try:
                loc = arvados.KeepLocator(word)
            except ValueError:
                continue  # this word isn't a locator, skip it
            if loc.md5sum not in locators_seen:
                locators_seen[loc.md5sum] = True
                total_bytes += loc.size

    return total_bytes

# copy_collection(obj_uuid, src, dst, args)
#
#    Copies the collection identified by obj_uuid from src to dst.
#    Returns the collection object created at dst.
#
#    If args.progress is True, produce a human-friendly progress
#    report.
#
#    If a collection with the desired portable_data_hash already
#    exists at dst, and args.force is False, copy_collection returns
#    the existing collection without copying any blocks.  Otherwise
#    (if no collection exists or if args.force is True)
#    copy_collection copies all of the collection data blocks from src
#    to dst.
#
#    For this application, it is critical to preserve the
#    collection's manifest hash, which is not guaranteed with the
#    arvados.CollectionReader and arvados.CollectionWriter classes.
#    Copying each block in the collection manually, followed by
#    the manifest block, ensures that the collection's manifest
#    hash will not change.
#
def copy_collection(obj_uuid, src, dst, args):
    c = src.collections().get(uuid=obj_uuid).execute()

    # If a collection with this hash already exists at the
    # destination, and 'force' is not true, just return that
    # collection.
    if not args.force:
        if 'portable_data_hash' in c:
            colhash = c['portable_data_hash']
        else:
            colhash = c['uuid']
        dstcol = dst.collections().list(
            filters=[['portable_data_hash', '=', colhash]]
        ).execute()
        if dstcol['items_available'] > 0:
            logger.debug("Skipping collection %s (already at dst)", obj_uuid)
            return dstcol['items'][0]

    # Fetch the collection's manifest.
    manifest = c['manifest_text']
    logger.debug("Copying collection %s with manifest: <%s>", obj_uuid, manifest)

    # Copy each block from src_keep to dst_keep.
    # Use the newly signed locators returned from dst_keep to build
    # a new manifest as we go.
    src_keep = arvados.keep.KeepClient(api_client=src, num_retries=2)
    dst_keep = arvados.keep.KeepClient(api_client=dst, num_retries=2)
    dst_manifest = ""
    dst_locators = {}
    bytes_written = 0
    bytes_expected = total_collection_size(manifest)
    if args.progress:
        progress_writer = ProgressWriter(human_progress)
    else:
        progress_writer = None

    for line in manifest.splitlines(True):
        words = line.split()
        dst_manifest_line = words[0]
        for word in words[1:]:
            try:
                loc = arvados.KeepLocator(word)
                blockhash = loc.md5sum
                # copy this block if we haven't seen it before
                # (otherwise, just reuse the existing dst_locator)
                if blockhash not in dst_locators:
                    logger.debug("Copying block %s (%s bytes)", blockhash, loc.size)
                    if progress_writer:
                        progress_writer.report(obj_uuid, bytes_written, bytes_expected)
                    data = src_keep.get(word)
                    dst_locator = dst_keep.put(data)
                    dst_locators[blockhash] = dst_locator
                    bytes_written += loc.size
                dst_manifest_line += ' ' + dst_locators[blockhash]
            except ValueError:
                # If 'word' can't be parsed as a locator,
                # presume it's a filename.
                dst_manifest_line += ' ' + word
        dst_manifest += dst_manifest_line
        if line.endswith("\n"):
            dst_manifest += "\n"

    if progress_writer:
        progress_writer.finish()

    # Copy the manifest and save the collection.
    logger.debug('saving %s with manifest: <%s>', obj_uuid, dst_manifest)
    dst_keep.put(dst_manifest)

    if 'uuid' in c:
        del c['uuid']
    if 'owner_uuid' in c:
        del c['owner_uuid']
    c['manifest_text'] = dst_manifest
    return dst.collections().create(body=c, ensure_unique_name=True).execute()

# copy_git_repo(src_git_repo, src, dst, dst_git_repo, script_version)
#
#    Copies commits from git repository 'src_git_repo' on Arvados
#    instance 'src' to 'dst_git_repo' on 'dst'.  Both src_git_repo
#    and dst_git_repo are repository names, not UUIDs (i.e. "arvados"
#    or "jsmith")
#
#    All commits will be copied to a destination branch named for the
#    source repository URL.
#
#    Because users cannot create their own repositories, the
#    destination repository must already exist.
#
#    The user running this command must be authenticated
#    to both repositories.
#
def copy_git_repo(src_git_repo, src, dst, dst_git_repo, script_version):
    # Identify the fetch and push URLs for the git repositories.
    r = src.repositories().list(
        filters=[['name', '=', src_git_repo]]).execute()
    if r['items_available'] != 1:
        raise Exception('cannot identify source repo {}; {} repos found'
                        .format(src_git_repo, r['items_available']))
    src_git_url = r['items'][0]['fetch_url']
    logger.debug('src_git_url: {}'.format(src_git_url))

    r = dst.repositories().list(
        filters=[['name', '=', dst_git_repo]]).execute()
    if r['items_available'] != 1:
        raise Exception('cannot identify destination repo {}; {} repos found'
                        .format(dst_git_repo, r['items_available']))
    dst_git_push_url  = r['items'][0]['push_url']
    logger.debug('dst_git_push_url: {}'.format(dst_git_push_url))

    # script_version is the "script_version" parameter from the source
    # component or job.  It is used here to tie the destination branch
    # to the commit that was used on the source.  If no script_version
    # was supplied in the component or job, it is a mistake in the pipeline,
    # but for the purposes of copying the repository, default to "master".
    #
    if not script_version:
        script_version = "master"

    dst_branch = re.sub(r'\W+', '_', "{}_{}".format(src_git_url, script_version))

    # Copy git commits from src repo to dst repo (but only if
    # we have not already copied this repo in this session).
    #
    if src_git_repo in local_repo_dir:
        logger.debug('already copied src repo %s, skipping', src_git_repo)
    else:
        tmprepo = tempfile.mkdtemp()
        local_repo_dir[src_git_repo] = tmprepo
        arvados.util.run_command(
            ["git", "clone", "--bare", src_git_url, tmprepo],
            cwd=os.path.dirname(tmprepo))
        arvados.util.run_command(
            ["git", "branch", dst_branch, script_version],
            cwd=tmprepo)
        arvados.util.run_command(["git", "remote", "add", "dst", dst_git_push_url], cwd=tmprepo)
        arvados.util.run_command(["git", "push", "dst", dst_branch], cwd=tmprepo)

# git_rev_parse(rev, repo)
#
#    Returns the 40-character commit hash corresponding to 'rev' in
#    git repository 'repo' (which must be the path of a local git
#    repository)
#
def git_rev_parse(rev, repo):
    gitout, giterr = arvados.util.run_command(
        ['git', 'rev-parse', rev], cwd=repo)
    return gitout.strip()

# uuid_type(api, object_uuid)
#
#    Returns the name of the class that object_uuid belongs to, based on
#    the second field of the uuid.  This function consults the api's
#    schema to identify the object class.
#
#    It returns a string such as 'Collection', 'PipelineInstance', etc.
#
#    Special case: if handed a Keep locator hash, return 'Collection'.
#
def uuid_type(api, object_uuid):
    if re.match(r'^[a-f0-9]{32}\+[0-9]+(\+[A-Za-z0-9+-]+)?$', object_uuid):
        return 'Collection'
    p = object_uuid.split('-')
    if len(p) == 3:
        type_prefix = p[1]
        for k in api._schema.schemas:
            obj_class = api._schema.schemas[k].get('uuidPrefix', None)
            if type_prefix == obj_class:
                return k
    return None

def abort(msg, code=1):
    logger.info("arv-copy:", msg)
    exit(code)


# Code for reporting on the progress of a collection upload.
# Stolen from arvados.commands.put.ArvPutCollectionWriter
# TODO(twp): figure out how to refactor into a shared library
# (may involve refactoring some arvados.commands.copy.copy_collection
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
