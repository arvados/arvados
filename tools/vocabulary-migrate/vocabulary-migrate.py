#!/usr/bin/env python3
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

import argparse
import copy
import json
import logging
import os
import sys

import arvados
import arvados.util

logger = logging.getLogger('arvados.vocabulary_migrate')
logger.setLevel(logging.INFO)

class VocabularyError(Exception):
    pass

opts = argparse.ArgumentParser(add_help=False)
opts.add_argument('--vocabulary-file', type=str, metavar='PATH', required=True,
                  help="""
Use vocabulary definition file at PATH for migration decisions.
""")
opts.add_argument('--dry-run', action='store_true', default=False,
                  help="""
Don't actually migrate properties, but only check if any collection/project
should be migrated.
""")
opts.add_argument('--debug', action='store_true', default=False,
                  help="""
Sets logging level to DEBUG.
""")
arg_parser = argparse.ArgumentParser(
    description='Migrate collections & projects properties to the new vocabulary format.',
    parents=[opts])

def parse_arguments(arguments):
    args = arg_parser.parse_args(arguments)
    if args.debug:
        logger.setLevel(logging.DEBUG)
    if not os.path.isfile(args.vocabulary_file):
        arg_parser.error("{} doesn't exist or isn't a file.".format(args.vocabulary_file))
    return args

def _label_to_id_mappings(data, obj_name):
    result = {}
    for obj_id, obj_data in data.items():
        for lbl in obj_data['labels']:
            obj_lbl = lbl['label']
            if obj_lbl not in result:
                result[obj_lbl] = obj_id
            else:
                raise VocabularyError('{} label "{}" for {} ID "{}" already seen at {} ID "{}".'.format(obj_name, obj_lbl, obj_name, obj_id, obj_name, result[obj_lbl]))
    return result

def key_labels_to_ids(vocab):
    return _label_to_id_mappings(vocab['tags'], 'key')

def value_labels_to_ids(vocab, key_id):
    if key_id in vocab['tags'] and 'values' in vocab['tags'][key_id]:
        return _label_to_id_mappings(vocab['tags'][key_id]['values'], 'value')
    return {}

def migrate_properties(properties, key_map, vocab):
    result = {}
    for k, v in properties.items():
        key = key_map.get(k, k)
        value = value_labels_to_ids(vocab, key).get(v, v)
        result[key] = value
    return result

def main(arguments=None):
    args = parse_arguments(arguments)
    vocab = None
    with open(args.vocabulary_file, 'r') as f:
        vocab = json.load(f)
    arv = arvados.api('v1')
    if 'tags' not in vocab or vocab['tags'] == {}:
        logger.warning('Empty vocabulary file, exiting.')
        return 1
    if not arv.users().current().execute()['is_admin']:
        logger.error('Admin privileges required.')
        return 1
    key_label_to_id_map = key_labels_to_ids(vocab)
    migrated_counter = 0

    for key_label in key_label_to_id_map:
        logger.debug('Querying objects with property key "{}"'.format(key_label))
        for resource in [arv.collections(), arv.groups()]:
            objs = arvados.util.keyset_list_all(
                resource.list,
                order='created_at',
                select=['uuid', 'properties'],
                filters=[['properties', 'exists', key_label]]
            )
            for o in objs:
                props = copy.copy(o['properties'])
                migrated_props = migrate_properties(props, key_label_to_id_map, vocab)
                if not args.dry_run:
                    logger.debug('Migrating {}: {} -> {}'.format(o['uuid'], props, migrated_props))
                    arv.collections().update(uuid=o['uuid'], body={
                        'properties': migrated_props
                    }).execute()
                else:
                    logger.info('Should migrate {}: {} -> {}'.format(o['uuid'], props, migrated_props))
                migrated_counter += 1
                if not args.dry_run and migrated_counter % 100 == 0:
                    logger.info('Migrating {} objects...'.format(migrated_counter))

    if args.dry_run and migrated_counter == 0:
        logger.info('Nothing to do.')
    elif not args.dry_run:
        logger.info('Done, total objects migrated: {}.'.format(migrated_counter))
    return 0

if __name__ == "__main__":
    sys.exit(main())
