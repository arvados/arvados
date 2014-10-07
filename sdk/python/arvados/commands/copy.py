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

def main():
    parser = argparse.ArgumentParser(
        description='Copy a pipeline instance from one Arvados instance to another.')

    parser.add_argument(
        '-v', '--verbose', dest='verbose', action='store_true',
        help='Verbose output.')
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
                                 force=args.force)
    elif t == 'PipelineInstance':
        result = copy_pipeline_instance(args.object_uuid,
                                        src_arv, dst_arv,
                                        args.dst_git_repo,
                                        dst_project=args.project_uuid,
                                        recursive=args.recursive)
    elif t == 'PipelineTemplate':
        result = copy_pipeline_template(args.object_uuid,
                                        src_arv, dst_arv,
                                        args.dst_git_repo,
                                        recursive=args.recursive)
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

# copy_pipeline_instance(pi_uuid, dst_git_repo, dst_project, recursive, src, dst)
#
#    Copies a pipeline instance identified by pi_uuid from src to dst.
#
#    If the 'recursive' option evaluates to True:
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
def copy_pipeline_instance(pi_uuid, src, dst, dst_git_repo, dst_project=None, recursive=True):
    # Fetch the pipeline instance record.
    pi = src.pipeline_instances().get(uuid=pi_uuid).execute()

    if recursive:
        if not dst_git_repo:
            abort('--dst-git-repo is required when copying a pipeline recursively.')
        # Copy the pipeline template and save the copied template.
        if pi.get('pipeline_template_uuid', None):
            pt = copy_pipeline_template(pi['pipeline_template_uuid'],
                                        src, dst,
                                        dst_git_repo,
                                        recursive=True)

        # Copy input collections, docker images and git repos.
        pi = copy_collections(pi, src, dst)
        copy_git_repos(pi, src, dst, dst_git_repo)

        # Update the fields of the pipeline instance with the copied
        # pipeline template.
        if pi.get('pipeline_template_uuid', None):
            pi['pipeline_template_uuid'] = pt['uuid']

    else:
        # not recursive
        print >>sys.stderr, "Copying only pipeline instance {}.".format(pi_uuid)
        print >>sys.stderr, "You are responsible for making sure all pipeline dependencies have been updated."

    # Update the pipeline instance properties, and create the new
    # instance at dst.
    pi['properties']['copied_from_pipeline_instance_uuid'] = pi_uuid
    pi['description'] = "Pipeline copied from {}\n\n{}".format(
        pi_uuid, pi.get('description', ''))
    if dst_project:
        pi['owner_uuid'] = dst_project
    else:
        del pi['owner_uuid']
    del pi['uuid']
    pi['ensure_unique_name'] = True

    new_pi = dst.pipeline_instances().create(body=pi).execute()
    return new_pi

# copy_pipeline_template(pt_uuid, src, dst, dst_git_repo, recursive)
#
#    Copies a pipeline template identified by pt_uuid from src to dst.
#
#    If the 'recursive' option evaluates to true, also copy any collections,
#    docker images and git repositories that this template references.
#
#    The owner_uuid of the new template is changed to that of the user
#    who copied the template.
#
#    Returns the copied pipeline template object.
#
def copy_pipeline_template(pt_uuid, src, dst, dst_git_repo, recursive=True):
    # fetch the pipeline template from the source instance
    pt = src.pipeline_templates().get(uuid=pt_uuid).execute()

    if recursive:
        if not dst_git_repo:
            abort('--dst-git-repo is required when copying a pipeline recursively.')
        # Copy input collections, docker images and git repos.
        pt = copy_collections(pt, src, dst)
        copy_git_repos(pt, src, dst, dst_git_repo)

    pt['description'] = "Pipeline template copied from {}\n\n{}".format(
        pt_uuid, pt.get('description', ''))
    pt['name'] = "{} copied from {}".format(pt.get('name', ''), pt_uuid)
    pt['ensure_unique_name'] = True
    del pt['uuid']
    del pt['owner_uuid']

    return dst.pipeline_templates().create(body=pt).execute()

# copy_collections(obj, src, dst)
#
#    Recursively copies all collections referenced by 'obj' from src
#    to dst.
#
#    Returns a copy of obj with any old collection uuids replaced by
#    the new ones.
#
def copy_collections(obj, src, dst):
    if type(obj) in [str, unicode]:
        if uuid_type(src, obj) == 'Collection':
            newc = copy_collection(obj, src, dst)
            if obj != newc['uuid'] and obj != newc['portable_data_hash']:
                return newc['uuid']
        return obj
    elif type(obj) == dict:
        return {v: copy_collections(obj[v], src, dst) for v in obj}
    elif type(obj) == list:
        return [copy_collections(v, src, dst) for v in obj]
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
            if repo not in copied:
                copy_git_repo(repo, src, dst, dst_repo)
                copied.add(repo)
            component['repository'] = dst_repo
            if 'script_version' in component:
                repo_dir = local_repo_dir[repo]
                component['script_version'] = git_rev_parse(component['script_version'], repo_dir)
        if 'job' in component:
            j = component['job']
            if 'repository' in j:
                repo = j['repository']
                if repo not in copied:
                    copy_git_repo(repo, src, dst, dst_repo)
                    copied.add(repo)
                j['repository'] = dst_repo
                repo_dir = local_repo_dir[repo]
                if 'script_version' in j:
                    j['script_version'] = git_rev_parse(j['script_version'], repo_dir)
                if 'supplied_script_version' in j:
                    j['supplied_script_version'] = git_rev_parse(j['supplied_script_version'], repo_dir)

# copy_collection(obj_uuid, src, dst)
#
#    Copies the collection identified by obj_uuid from src to dst.
#    Returns the collection object created at dst.
#
#    For this application, it is critical to preserve the
#    collection's manifest hash, which is not guaranteed with the
#    arvados.CollectionReader and arvados.CollectionWriter classes.
#    Copying each block in the collection manually, followed by
#    the manifest block, ensures that the collection's manifest
#    hash will not change.
#
def copy_collection(obj_uuid, src, dst, force=False):
    c = src.collections().get(uuid=obj_uuid).execute()

    # If a collection with this hash already exists at the
    # destination, and 'force' is not true, just return that
    # collection.
    if not force:
        if 'portable_data_hash' in c:
            colhash = c['portable_data_hash']
        else:
            colhash = c['uuid']
        dstcol = dst.collections().list(
            filters=[['portable_data_hash', '=', colhash]]
        ).execute()
        if dstcol['items_available'] > 0:
            logger.info("Skipping collection %s (already at dst)", obj_uuid)
            return dstcol['items'][0]

    logger.info("Copying collection %s", obj_uuid)

    # Fetch the collection's manifest.
    manifest = c['manifest_text']

    # Enumerate the block locators found in the manifest.
    collection_blocks = set()
    src_keep = arvados.keep.KeepClient(api_client=src, num_retries=2)
    for line in manifest.splitlines():
        for block_hash in line.split()[1:]:
            if arvados.util.portable_data_hash_pattern.match(block_hash):
                collection_blocks.add(block_hash)
            else:
                break

    # Copy each block from src_keep to dst_keep.
    dst_keep = arvados.keep.KeepClient(api_client=dst, num_retries=2)
    for locator in collection_blocks:
        parts = locator.split('+')
        logger.info("Copying block %s (%s bytes)", locator, parts[1])
        data = src_keep.get(locator)
        dst_keep.put(data)

    # Copy the manifest and save the collection.
    logger.debug('saving {} manifest: {}'.format(obj_uuid, manifest))
    dst_keep.put(manifest)

    if 'uuid' in c:
        del c['uuid']
    if 'owner_uuid' in c:
        del c['owner_uuid']
    c['ensure_unique_name'] = True
    return dst.collections().create(body=c).execute()

# copy_git_repo(src_git_repo, src, dst, dst_git_repo)
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
def copy_git_repo(src_git_repo, src, dst, dst_git_repo):
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

    dst_branch = re.sub(r'\W+', '_', src_git_url)

    # Copy git commits from src repo to dst repo (but only if
    # we have not already copied this repo in this session).
    #
    if src_git_repo in local_repo_dir:
        logger.debug('already copied src repo %s, skipping', src_git_repo)
    else:
        tmprepo = tempfile.mkdtemp()
        local_repo_dir[src_git_repo] = tmprepo
        arvados.util.run_command(
            ["git", "clone", src_git_url, tmprepo],
            cwd=os.path.dirname(tmprepo))
        arvados.util.run_command(
            ["git", "checkout", "-b", dst_branch],
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
    print >>sys.stderr, "arv-copy:", msg
    exit(code)

if __name__ == '__main__':
    main()
