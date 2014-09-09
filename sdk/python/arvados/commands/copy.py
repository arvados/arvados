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
        help='Recursively add any objects that this object depends upon.')
    parser.add_argument(
        '--no-recursive', dest='recursive', action='store_false')
    parser.add_argument(
        '--dest-git-repo', dest='dest_git_repo',
        help='The name of the destination git repository.')
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
        result = copy_pipeline_instance(args.object_uuid, args.dest_git_repo, src=src_arv, dst=dst_arv)
    elif t == 'PipelineTemplate':
        result = copy_pipeline_template(args.object_uuid, src=src_arv, dst=dst_arv)
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

# copy_collection(obj_uuid, src, dst)
#
#    Copy the collection identified by obj_uuid from src to dst.
#
#    For this application, it is critical to preserve the
#    collection's manifest hash, which is not guaranteed with the
#    arvados.CollectionReader and arvados.CollectionWriter classes.
#    Copying each block in the collection manually, followed by
#    the manifest block, ensures that the collection's manifest
#    hash will not change.
#
def copy_collection(obj_uuid, src=None, dst=None):
    # Fetch the collection's manifest.
    c = src.collections().get(uuid=obj_uuid).execute()
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

# copy_pipeline_instance(obj_uuid, dst_git_repo, src, dst)
#
#    Copies a pipeline instance identified by obj_uuid from src to dst.
#
#    If the recursive option is on:
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
def copy_pipeline_instance(obj_uuid, dst_git_repo, src=None, dst=None):
    # Fetch the pipeline instance record.
    pi = src.pipeline_instances().get(uuid=obj_uuid).execute()

    # Copy input collections and docker images:
    # For each component c in the pipeline, add any
    # collection hashes found in c['job']['dependencies']
    # and c['job']['docker_image_locator'].
    #
    input_collections = sets.Set()
    for cname in pi['components']:
        if 'job' not in pi['components'][cname]:
            continue
        job = pi['components'][cname]['job']
        for dep in job['dependencies']:
            input_collections.add(dep)
        docker = job.get('docker_image_locator', None)
        if docker:
            input_collections.add(docker)

    for c in input_collections:
        copy_collection(c, src, dst)

    # Copy the git repositorie(s)
    repos = sets.Set()
    for c in pi['components']:
        component = pi['components'][c]
        if 'repository' in component:
            repos.add(component['repository'])
        if 'job' in component and 'repository' in component['job']:
            repos.add(component['job']['repository'])

    for r in repos:
        dst_branch = '{}_{}'.format(obj_uuid, r)
        copy_git_repo(r, dst_git_repo, dst_branch, src, dst)

    # Copy the pipeline template and save the uuid of the copy
    new_pt = copy_pipeline_template(pi['pipeline_template_uuid'], src, dst)

    # Update the fields of the pipeline instance
    pi['properties']['copied_from_pipeline_instance_uuid'] = obj_uuid
    pi['pipeline_template_uuid'] = new_pt
    del pi['owner_uuid']

    # Rename the repositories named in the components to the dst_git_repo
    for c in pi['components']:
        component = pi['components'][c]
        if 'repository' in component:
            component['repository'] = dst_git_repo
        if 'job' in component and 'repository' in component['job']:
            component['job']['repository'] = dst_git_repo

    # Create the new pipeline instance at the destination Arvados.
    new_pi = dst.pipeline_instances().create(pipeline_instance=pi).execute()
    return new_pi

# copy_pipeline_template(obj_uuid, src, dst)
#
#    Copies a pipeline template identified by obj_uuid from src to dst.
#
#    The owner_uuid of the new template is changed to that of the user
#    who copied the template.
#
def copy_pipeline_template(obj_uuid, src=None, dst=None):
    # fetch the pipeline template from the source instance
    old_pt = src.pipeline_templates().get(uuid=obj_uuid).execute()
    old_pt['name'] = old_pt['name'] + ' copy'
    del old_pt['uuid']
    del old_pt['owner_uuid']
    return dst.pipeline_templates().create(body=old_pt).execute()

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
