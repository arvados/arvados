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
import sets
import sys
import logging

import arvados
import arvados.config
import arvados.keep

def main():
    logger = logging.getLogger('arvados.arv-copy')
    logger.setLevel(logging.DEBUG)

    parser = argparse.ArgumentParser(
        description='Copy a pipeline instance from one Arvados instance to another.')

    parser.add_argument(
        '--recursive', dest='recursive', action='store_true',
        help='Recursively add any objects that this object depends upon.')
    parser.add_argument(
        '--no-recursive', dest='recursive', action='store_false')
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
        result = copy_pipeline_instance(args.object_uuid, src=src_arv, dst=dst_arv)
    elif t == 'PipelineTemplate':
        result = copy_pipeline_template(args.object_uuid, src=src_arv, dst=dst_arv)
    else:
        abort("cannot copy object {} of type {}".format(args.object_uuid, t))

    print result
    exit(0)

# Creates an API client for the Arvados instance identified by
# instance_name.  Looks in $HOME/.config/arvados/instance_name.conf
# for credentials.
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
    for locator in collection_blocks:
        data = src_keep.get(locator)
        logger.debug('copying block %s', locator)
        logger.info("Retrieved %d bytes", len(data))
        dst_keep.put(data)

    # Copy the manifest and save the collection.
    dst_keep.put(manifest)
    return dst_keep.collections().create(manifest_text=manifest).execute()

# copy_pipeline_instance(obj_uuid, src, dst)
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
def copy_pipeline_instance(obj_uuid, src=None, dst=None):
    # Fetch the pipeline instance record.
    pi = src.pipeline_instances().get(uuid=obj_uuid).execute()

    # Copy input collections and docker images:
    # For each component c in the pipeline, add any
    # collection hashes found in c['job']['dependencies']
    # and c['job']['docker_image_locator'].
    #
    input_collections = sets.Set()
    for cname in pi['components']:
        job = pi['components'][cname]['job']
        for dep in job['dependencies']:
            input_collections.add(dep)
        docker = job.get('docker_image_locator', None)
        if docker:
            input_collections.add(docker)

    for c in input_collections:
        copy_collection(c, src, dst)

    # Copy the pipeline template and save the uuid of the copy
    new_pt = copy_pipeline_template(pi['pipeline_template_uuid'], src, dst)

    # Update the fields of the pipeline instance
    pi['properties']['copied_from_pipeline_instance_uuid'] = obj_uuid
    pi['pipeline_template_uuid'] = new_pt
    del pi['owner_uuid']

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

# uuid_type(api, object_uuid)
#
#    Returns the name of the class that object_uuid belongs to, based on
#    the second field of the uuid.  This function consults the api's
#    schema to identify the object class.
#
#    It returns a string such as 'Collection', 'PipelineInstance', etc.
#
def uuid_type(api, object_uuid):
    type_prefix = object_uuid.split('-')[1]
    for k in api._schema.schemas:
        obj_class = api._schema.schemas[k].get('uuidPrefix', None)
        if obj_class:
            return obj_class
    return None

def abort(msg, code=1):
    print >>sys.stderr, "arv-copy:", msg
    exit(code)

if __name__ == '__main__':
    main()
