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
import arvados.commands._util as arv_cmd
import arvados.commands.keepdocker

from arvados.api import OrderedJsonModel

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

# Set of (repository, script_version) two-tuples of commits copied in git.
scripts_copied = set()

# The owner_uuid of the object being copied
src_owner_uuid = None

def main():
    copy_opts = argparse.ArgumentParser(add_help=False)

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
        '--src', dest='source_arvados', required=True,
        help='The name of the source Arvados instance (required). May be either a pathname to a config file, or the basename of a file in $HOME/.config/arvados/instance_name.conf.')
    copy_opts.add_argument(
        '--dst', dest='destination_arvados', required=True,
        help='The name of the destination Arvados instance (required). May be either a pathname to a config file, or the basename of a file in $HOME/.config/arvados/instance_name.conf.')
    copy_opts.add_argument(
        '--recursive', dest='recursive', action='store_true',
        help='Recursively copy any dependencies for this object. (default)')
    copy_opts.add_argument(
        '--no-recursive', dest='recursive', action='store_false',
        help='Do not copy any dependencies. NOTE: if this option is given, the copied object will need to be updated manually in order to be functional.')
    copy_opts.add_argument(
        '--dst-git-repo', dest='dst_git_repo',
        help='The name of the destination git repository. Required when copying a pipeline recursively.')
    copy_opts.add_argument(
        '--project-uuid', dest='project_uuid',
        help='The UUID of the project at the destination to which the pipeline should be copied.')
    copy_opts.add_argument(
        'object_uuid',
        help='The UUID of the object to be copied.')
    copy_opts.set_defaults(progress=True)
    copy_opts.set_defaults(recursive=True)

    parser = argparse.ArgumentParser(
        description='Copy a pipeline instance, template or collection from one Arvados instance to another.',
        parents=[copy_opts, arv_cmd.retry_opt])
    args = parser.parse_args()

    if args.verbose:
        logger.setLevel(logging.DEBUG)
    else:
        logger.setLevel(logging.INFO)

    # Create API clients for the source and destination instances
    src_arv = api_for_instance(args.source_arvados)
    dst_arv = api_for_instance(args.destination_arvados)

    if not args.project_uuid:
        args.project_uuid = dst_arv.users().current().execute(num_retries=args.retries)["uuid"]

    # Identify the kind of object we have been given, and begin copying.
    t = uuid_type(src_arv, args.object_uuid)
    if t == 'Collection':
        set_src_owner_uuid(src_arv.collections(), args.object_uuid, args)
        result = copy_collection(args.object_uuid,
                                 src_arv, dst_arv,
                                 args)
    elif t == 'PipelineInstance':
        set_src_owner_uuid(src_arv.pipeline_instances(), args.object_uuid, args)
        result = copy_pipeline_instance(args.object_uuid,
                                        src_arv, dst_arv,
                                        args)
    elif t == 'PipelineTemplate':
        set_src_owner_uuid(src_arv.pipeline_templates(), args.object_uuid, args)
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

def set_src_owner_uuid(resource, uuid, args):
    global src_owner_uuid
    c = resource.get(uuid=uuid).execute(num_retries=args.retries)
    src_owner_uuid = c.get("owner_uuid")

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
                             model=OrderedJsonModel())
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
    pi = src.pipeline_instances().get(uuid=pi_uuid).execute(num_retries=args.retries)

    if args.recursive:
        if not args.dst_git_repo:
            abort('--dst-git-repo is required when copying a pipeline recursively.')
        # Copy the pipeline template and save the copied template.
        if pi.get('pipeline_template_uuid', None):
            pt = copy_pipeline_template(pi['pipeline_template_uuid'],
                                        src, dst, args)

        # Copy input collections, docker images and git repos.
        pi = copy_collections(pi, src, dst, args)
        copy_git_repos(pi, src, dst, args.dst_git_repo, args)
        copy_docker_images(pi, src, dst, args)

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
        pi_uuid,
        pi['description'] if pi.get('description', None) else '')

    pi['owner_uuid'] = args.project_uuid

    del pi['uuid']

    new_pi = dst.pipeline_instances().create(body=pi, ensure_unique_name=True).execute(num_retries=args.retries)
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
    pt = src.pipeline_templates().get(uuid=pt_uuid).execute(num_retries=args.retries)

    if args.recursive:
        if not args.dst_git_repo:
            abort('--dst-git-repo is required when copying a pipeline recursively.')
        # Copy input collections, docker images and git repos.
        pt = copy_collections(pt, src, dst, args)
        copy_git_repos(pt, src, dst, args.dst_git_repo, args)
        copy_docker_images(pt, src, dst, args)

    pt['description'] = "Pipeline template copied from {}\n\n{}".format(
        pt_uuid,
        pt['description'] if pt.get('description', None) else '')
    pt['name'] = "{} copied from {}".format(pt.get('name', ''), pt_uuid)
    del pt['uuid']

    pt['owner_uuid'] = args.project_uuid

    return dst.pipeline_templates().create(body=pt, ensure_unique_name=True).execute(num_retries=args.retries)

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

def migrate_jobspec(jobspec, src, dst, dst_repo, args):
    """Copy a job's script to the destination repository, and update its record.

    Given a jobspec dictionary, this function finds the referenced script from
    src and copies it to dst and dst_repo.  It also updates jobspec in place to
    refer to names on the destination.
    """
    repo = jobspec.get('repository')
    if repo is None:
        return
    # script_version is the "script_version" parameter from the source
    # component or job.  If no script_version was supplied in the
    # component or job, it is a mistake in the pipeline, but for the
    # purposes of copying the repository, default to "master".
    script_version = jobspec.get('script_version') or 'master'
    script_key = (repo, script_version)
    if script_key not in scripts_copied:
        copy_git_repo(repo, src, dst, dst_repo, script_version, args)
        scripts_copied.add(script_key)
    jobspec['repository'] = dst_repo
    repo_dir = local_repo_dir[repo]
    for version_key in ['script_version', 'supplied_script_version']:
        if version_key in jobspec:
            jobspec[version_key] = git_rev_parse(jobspec[version_key], repo_dir)

# copy_git_repos(p, src, dst, dst_repo, args)
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
def copy_git_repos(p, src, dst, dst_repo, args):
    for component in p['components'].itervalues():
        migrate_jobspec(component, src, dst, dst_repo, args)
        if 'job' in component:
            migrate_jobspec(component['job'], src, dst, dst_repo, args)

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

def create_collection_from(c, src, dst, args):
    """Create a new collection record on dst, and copy Docker metadata if
    available."""

    collection_uuid = c['uuid']
    del c['uuid']

    if not c["name"]:
        c['name'] = "copied from " + collection_uuid

    if 'properties' in c:
        del c['properties']

    c['owner_uuid'] = args.project_uuid

    dst_collection = dst.collections().create(body=c, ensure_unique_name=True).execute(num_retries=args.retries)

    # Create docker_image_repo+tag and docker_image_hash links
    # at the destination.
    for link_class in ("docker_image_repo+tag", "docker_image_hash"):
        docker_links = src.links().list(filters=[["head_uuid", "=", collection_uuid], ["link_class", "=", link_class]]).execute(num_retries=args.retries)['items']

        for src_link in docker_links:
            body = {key: src_link[key]
                    for key in ['link_class', 'name', 'properties']}
            body['head_uuid'] = dst_collection['uuid']
            body['owner_uuid'] = args.project_uuid

            lk = dst.links().create(body=body).execute(num_retries=args.retries)
            logger.debug('created dst link {}'.format(lk))

    return dst_collection

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
    if arvados.util.keep_locator_pattern.match(obj_uuid):
        # If the obj_uuid is a portable data hash, it might not be uniquely
        # identified with a particular collection.  As a result, it is
        # ambigious as to what name to use for the copy.  Apply some heuristics
        # to pick which collection to get the name from.
        srccol = src.collections().list(
            filters=[['portable_data_hash', '=', obj_uuid]],
            order="created_at asc"
            ).execute(num_retries=args.retries)

        items = srccol.get("items")

        if not items:
            logger.warning("Could not find collection with portable data hash %s", obj_uuid)
            return

        c = None

        if len(items) == 1:
            # There's only one collection with the PDH, so use that.
            c = items[0]
        if not c:
            # See if there is a collection that's in the same project
            # as the root item (usually a pipeline) being copied.
            for i in items:
                if i.get("owner_uuid") == src_owner_uuid and i.get("name"):
                    c = i
                    break
        if not c:
            # Didn't find any collections located in the same project, so
            # pick the oldest collection that has a name assigned to it.
            for i in items:
                if i.get("name"):
                    c = i
                    break
        if not c:
            # None of the collections have names (?!), so just pick the
            # first one.
            c = items[0]

        # list() doesn't return manifest text (and we don't want it to,
        # because we don't need the same maninfest text sent to us 50
        # times) so go and retrieve the collection object directly
        # which will include the manifest text.
        c = src.collections().get(uuid=c["uuid"]).execute(num_retries=args.retries)
    else:
        # Assume this is an actual collection uuid, so fetch it directly.
        c = src.collections().get(uuid=obj_uuid).execute(num_retries=args.retries)

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
        ).execute(num_retries=args.retries)
        if dstcol['items_available'] > 0:
            for d in dstcol['items']:
                if ((args.project_uuid == d['owner_uuid']) and
                    (c.get('name') == d['name']) and
                    (c['portable_data_hash'] == d['portable_data_hash'])):
                    return d
            c['manifest_text'] = dst.collections().get(
                uuid=dstcol['items'][0]['uuid']
            ).execute(num_retries=args.retries)['manifest_text']
            return create_collection_from(c, src, dst, args)

    # Fetch the collection's manifest.
    manifest = c['manifest_text']
    logger.debug("Copying collection %s with manifest: <%s>", obj_uuid, manifest)

    # Copy each block from src_keep to dst_keep.
    # Use the newly signed locators returned from dst_keep to build
    # a new manifest as we go.
    src_keep = arvados.keep.KeepClient(api_client=src, num_retries=args.retries)
    dst_keep = arvados.keep.KeepClient(api_client=dst, num_retries=args.retries)
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
        progress_writer.report(obj_uuid, bytes_written, bytes_expected)
        progress_writer.finish()

    # Copy the manifest and save the collection.
    logger.debug('saving %s with manifest: <%s>', obj_uuid, dst_manifest)

    dst_keep.put(dst_manifest.encode('utf-8'))
    c['manifest_text'] = dst_manifest
    return create_collection_from(c, src, dst, args)

# copy_git_repo(src_git_repo, src, dst, dst_git_repo, script_version, args)
#
#    Copies commits from git repository 'src_git_repo' on Arvados
#    instance 'src' to 'dst_git_repo' on 'dst'.  Both src_git_repo
#    and dst_git_repo are repository names, not UUIDs (i.e. "arvados"
#    or "jsmith")
#
#    All commits will be copied to a destination branch named for the
#    source repository URL.
#
#    The destination repository must already exist.
#
#    The user running this command must be authenticated
#    to both repositories.
#
def copy_git_repo(src_git_repo, src, dst, dst_git_repo, script_version, args):
    # Identify the fetch and push URLs for the git repositories.
    r = src.repositories().list(
        filters=[['name', '=', src_git_repo]]).execute(num_retries=args.retries)
    if r['items_available'] != 1:
        raise Exception('cannot identify source repo {}; {} repos found'
                        .format(src_git_repo, r['items_available']))
    src_git_url = r['items'][0]['fetch_url']
    logger.debug('src_git_url: {}'.format(src_git_url))

    r = dst.repositories().list(
        filters=[['name', '=', dst_git_repo]]).execute(num_retries=args.retries)
    if r['items_available'] != 1:
        raise Exception('cannot identify destination repo {}; {} repos found'
                        .format(dst_git_repo, r['items_available']))
    dst_git_push_url  = r['items'][0]['push_url']
    logger.debug('dst_git_push_url: {}'.format(dst_git_push_url))

    dst_branch = re.sub(r'\W+', '_', "{}_{}".format(src_git_url, script_version))

    # Copy git commits from src repo to dst repo.
    if src_git_repo not in local_repo_dir:
        local_repo_dir[src_git_repo] = tempfile.mkdtemp()
        arvados.util.run_command(
            ["git", "clone", "--bare", src_git_url,
             local_repo_dir[src_git_repo]],
            cwd=os.path.dirname(local_repo_dir[src_git_repo]))
        arvados.util.run_command(
            ["git", "remote", "add", "dst", dst_git_push_url],
            cwd=local_repo_dir[src_git_repo])
    arvados.util.run_command(
        ["git", "branch", dst_branch, script_version],
        cwd=local_repo_dir[src_git_repo])
    arvados.util.run_command(["git", "push", "dst", dst_branch],
                             cwd=local_repo_dir[src_git_repo])

def copy_docker_images(pipeline, src, dst, args):
    """Copy any docker images named in the pipeline components'
    runtime_constraints field from src to dst."""

    logger.debug('copy_docker_images: {}'.format(pipeline['uuid']))
    for c_name, c_info in pipeline['components'].iteritems():
        if ('runtime_constraints' in c_info and
            'docker_image' in c_info['runtime_constraints']):
            copy_docker_image(
                c_info['runtime_constraints']['docker_image'],
                c_info['runtime_constraints'].get('docker_image_tag', 'latest'),
                src, dst, args)


def copy_docker_image(docker_image, docker_image_tag, src, dst, args):
    """Copy the docker image identified by docker_image and
    docker_image_tag from src to dst. Create appropriate
    docker_image_repo+tag and docker_image_hash links at dst.

    """

    logger.debug('copying docker image {}:{}'.format(docker_image, docker_image_tag))

    # Find the link identifying this docker image.
    docker_image_list = arvados.commands.keepdocker.list_images_in_arv(
        src, args.retries, docker_image, docker_image_tag)
    if docker_image_list:
        image_uuid, image_info = docker_image_list[0]
        logger.debug('copying collection {} {}'.format(image_uuid, image_info))

        # Copy the collection it refers to.
        dst_image_col = copy_collection(image_uuid, src, dst, args)
    elif arvados.util.keep_locator_pattern.match(docker_image):
        dst_image_col = copy_collection(docker_image, src, dst, args)
    else:
        logger.warning('Could not find docker image {}:{}'.format(docker_image, docker_image_tag))

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
