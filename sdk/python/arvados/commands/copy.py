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
import os
import re
import sets
import sys
import logging
import tempfile

import arvados
import arvados.config
import arvados.keep

logger = logging.getLogger('arvados.arv-copy')

def main():
    logger.setLevel(logging.DEBUG)

    parser = argparse.ArgumentParser(
        description='Copy a pipeline instance from one Arvados instance to another.')

    parser.add_argument(
        '--recursive', dest='recursive', action='store_true',
        help='Recursively copy any dependencies for this object. (default)')
    parser.add_argument(
        '--no-recursive', dest='recursive', action='store_false',
        help='Do not copy any dependencies. NOTE: if this option is given, the copied object will need to be updated manually in order to be functional.')
    parser.add_argument(
        '--dst-git-repo', dest='dst_git_repo',
        help='The name of the destination git repository.')
    parser.add_argument(
        '--project_uuid', dest='project_uuid',
        help='The UUID of the project at the destination to which the pipeline should be copied.')
    parser.add_argument(
        'object_uuid',
        help='The UUID of the object to be copied.')
    parser.add_argument(
        'source_arvados',
        help='The name of the source Arvados instance.')
    parser.add_argument(
        'destination_arvados',
        help='The name of the destination Arvados instance.')
    parser.set_defaults(recursive=True)

    args = parser.parse_args()

    # Create API clients for the source and destination instances
    src_arv = api_for_instance(args.source_arvados)
    dst_arv = api_for_instance(args.destination_arvados)

    # Identify the kind of object we have been given, and begin copying.
    t = uuid_type(src_arv, args.object_uuid)
    if t == 'Collection':
        result = copy_collection(args.object_uuid, src=src_arv, dst=dst_arv)
    elif t == 'PipelineInstance':
        result = copy_pipeline_instance(args.object_uuid,
                                        dst_git_repo=args.dst_git_repo,
                                        dst_project=args.project_uuid,
                                        recursive=args.recursive,
                                        src=src_arv, dst=dst_arv)
    elif t == 'PipelineTemplate':
        result = copy_pipeline_template(args.object_uuid,
                                        recursive=args.recursive,
                                        src=src_arv, dst=dst_arv)
    else:
        abort("cannot copy object {} of type {}".format(args.object_uuid, t))

    print result
    exit(0)

# api_for_instance(instance_name)
#
#     Creates an API client for the Arvados instance identified by
#     instance_name.  Credentials must be stored in
#     $HOME/.config/arvados/instance_name.conf
#
def api_for_instance(instance_name):
    if '/' in instance_name:
        abort('illegal instance name {}'.format(instance_name))
    config_file = os.path.join(os.environ['HOME'], '.config', 'arvados', "{}.conf".format(instance_name))
    cfg = arvados.config.load(config_file)

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
def copy_pipeline_instance(pi_uuid, dst_git_repo=None, dst_project=None, recursive=True, src=None, dst=None):
    # Fetch the pipeline instance record.
    pi = src.pipeline_instances().get(uuid=pi_uuid).execute()
    pi['properties']['copied_from_pipeline_instance_uuid'] = pi_uuid

    if recursive:
        # Copy the pipeline template and save the copied template.
        pt = copy_pipeline_template(pi['pipeline_template_uuid'],
                                    recursive=True,
                                    src, dst)

        # Copy input collections, docker images and git repos.
        pi = copy_collections(pi, src, dst)
        copy_git_repos(pi, dst_git_repo, src, dst)

        # Update the fields of the pipeline instance with the copied
        # pipeline template.
        pi['pipeline_template_uuid'] = pt['uuid']
        if dst_project:
            pi['owner_uuid'] = dst_project
        else:
            del pi['owner_uuid']

    else:
        # not recursive
        print >>sys.stderr, "Copying only pipeline instance {}.".format(pi_uuid)
        print >>sys.stderr, "You are responsible for making sure all pipeline dependencies have been updated."

    # Create the new pipeline instance at the destination Arvados.
    new_pi = dst.pipeline_instances().create(pipeline_instance=pi).execute()
    return new_pi

# copy_pipeline_template(pt_uuid, recursive, src, dst)
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
def copy_pipeline_template(pt_uuid, recursive=True, src=None, dst=None):
    # fetch the pipeline template from the source instance
    pt = src.pipeline_templates().get(uuid=pt_uuid).execute()

    if recursive:
        # Copy input collections, docker images and git repos.
        pt = copy_collections(pt, src, dst)
        copy_git_repos(pt, dst_git_repo, src, dst)

    pt['name'] = pt['name'] + ' copy'
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
    if type(obj) == str:
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

# copy_git_repos(p, dst_repo, dst_branch, src, dst)
#
#    Copy all git repositories referenced by pipeline instance or
#    template 'p' from src to dst.
#
#    p is updated 
#    Git repository dependencies are identified by:
#      * p['components'][c]['repository']
#      * p['components'][c]['job']['repository']
#    for each component c in the pipeline.
#
def copy_git_repos(p, dst_repo, src=None, dst=None):
    copied = set()
    for c in p['components']:
        component = p['components'][c]
        if 'repository' in component:
            repo = component['repository']
            if repo not in copied:
                dst_branch = p['uuid']
                copy_git_repo(repo, dst_repo, dst_branch, src, dst)
                copied.add(repo)
            component['repository'] = dst_repo
        if 'job' in component and 'repository' in component['job']:
            repo = component['job']['repository']
            if repo not in copied:
                dst_branch = p['uuid']
                copy_git_repo(repo, dst_repo, dst_branch, src, dst)
                copied.add(repo)
            component['job']['repository'] = dst_repo
    return repos

# copy_collection(obj_uuid, src, dst)
#
#    Copy the collection identified by obj_uuid from src to dst.
#    Returns the collection object created at dst.
#
#    For this application, it is critical to preserve the
#    collection's manifest hash, which is not guaranteed with the
#    arvados.CollectionReader and arvados.CollectionWriter classes.
#    Copying each block in the collection manually, followed by
#    the manifest block, ensures that the collection's manifest
#    hash will not change.
#
def copy_collection(obj_uuid, src=None, dst=None):
    c = src.collections().get(uuid=obj_uuid).execute()

    # Check whether a collection with this hash already exists
    # at the destination.  If so, just return that collection.
    if 'portable_data_hash' in c:
        colhash = c['portable_data_hash']
    else:
        colhash = c['uuid']
    dstcol = dst.collections().list(
        filters=[['portable_data_hash', '=', colhash]]
    ).execute()
    if dstcol['items_available'] > 0:
        return dstcol['items'][0]

    # Fetch the collection's manifest.
    manifest = c['manifest_text']
    logging.debug('copying collection %s', obj_uuid)
    logging.debug('manifest_text = %s', manifest)

    # Enumerate the block locators found in the manifest.
    collection_blocks = sets.Set()
    src_keep = arvados.keep.KeepClient(src)
    for line in manifest.splitlines():
        try:
            block_hash = line.split()[1]
            collection_blocks.add(block_hash)
        except ValueError:
            abort('bad manifest line in collection {}: {}'.format(obj_uuid, f))

    # Copy each block from src_keep to dst_keep.
    dst_keep = arvados.keep.KeepClient(dst)
    for locator in collection_blocks:
        data = src_keep.get(locator)
        logger.debug('copying block %s', locator)
        logger.info("Retrieved %d bytes", len(data))
        dst_keep.put(data)

    # Copy the manifest and save the collection.
    logger.debug('saving {} manifest: {}'.format(obj_uuid, manifest))
    dst_keep.put(manifest)
    return dst.collections().create(body={"manifest_text": manifest}).execute()

# copy_git_repo(src_git_repo, dst_git_repo, dst_branch, src, dst)
#
#    Copies commits from git repository 'src_git_repo' on Arvados
#    instance 'src' to 'dst_git_repo' on 'dst'. A branch 'dst_branch'
#    is created at the destination repository, and commits from
#    src_git_repo are merged onto that branch.
#
#    Because users cannot create their own repositories, the
#    destination repository must already exist.
#
#    The user running this command must be authenticated
#    to both repositories.
#
def copy_git_repo(src_git_repo, dst_git_repo, dst_branch, src=None, dst=None):
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
        raise Exception('cannot identify source repo {}; {} repos found'
                        .format(dst_git_repo, r['items_available']))
    dst_git_push_url  = r['items'][0]['push_url']
    logger.debug('dst_git_push_url: {}'.format(dst_git_push_url))

    tmprepo = tempfile.mkdtemp()

    arvados.util.run_command(
        ["git", "clone", src_git_url, tmprepo],
        cwd=os.path.dirname(tmprepo))
    arvados.util.run_command(
        ["git", "checkout", "-B", dst_branch],
        cwd=tmprepo)
    arvados.util.run_command(["git", "remote", "add", "dst", dst_git_push_url], cwd=tmprepo)
    arvados.util.run_command(["git", "push", "dst"], cwd=tmprepo)

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
    if re.match(r'^[a-f0-9]{32}(\+[A-Za-z0-9+-]+)?$', object_uuid):
        return 'Collection'
    type_prefix = object_uuid.split('-')[1]
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
