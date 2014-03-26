#! /usr/bin/env python

import arvados

import argparse
import pprint
import re

from collections import defaultdict
from math import log
from operator import itemgetter

arv = arvados.api('v1')

# Adapted from http://stackoverflow.com/questions/4180980/formatting-data-quantity-capacity-as-string
byteunits = ('B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB')
def fileSizeFormat(value):
  exponent = 0 if value == 0 else int(log(value, 1024))
  return "%7.2f %-3s" % (float(value) / pow(1024, exponent),
                         byteunits[exponent])

def byteSizeFromValidUuid(valid_uuid):
  return int(valid_uuid.split('+')[1])

class CollectionInfo:
  all_by_uuid = {}

  def __init__(self, uuid):
    if CollectionInfo.all_by_uuid.has_key(uuid):
      raise ValueError('Collection for uuid "%s" already exists.' % uuid)
    self.uuid = uuid
    self.block_uuids = set()  # uuids of keep blocks in this collection
    self.reader_uuids = set()  # uuids of users who can read this collection
    self.persister_uuids = set()  # uuids of users who want this collection saved
    CollectionInfo.all_by_uuid[uuid] = self

  def byte_size(self):
    return sum(map(byteSizeFromValidUuid, self.block_uuids))

  def __str__(self):
    return ('CollectionInfo uuid: %s\n'
            '               %d block(s) containing %s\n'
            '               reader_uuids: %s\n'
            '               persister_uuids: %s' %
            (self.uuid,
             len(self.block_uuids),
             fileSizeFormat(self.byte_size()),
             pprint.pformat(self.reader_uuids, indent = 15),
             pprint.pformat(self.persister_uuids, indent = 15)))

  @staticmethod
  def get(uuid):
    if not CollectionInfo.all_by_uuid.has_key(uuid):
      CollectionInfo(uuid)
    return CollectionInfo.all_by_uuid[uuid]
  

def nonNegativeIntegerFromString(s, base):
  """Returns the value s represents in the specified base as long as it is greater than zero, otherwise None."""
  try:
    value = int(s, base)
    if value >= 0:
      return value
  except ValueError:
    pass  # We'll cover this in the next line.
  return None

# TODO(misha): Add tests.
BLOCK_HASH_LENGTH = 32
def hashAndSizeFromBlockLocator(candidate):
  """Returns the [integer hash, integer size] from a block locator.

  The returned hash will be None iff this is not a valid block locator.
  The returned size will be None if this is not a valid block locator
  or it is a valid block locator which does not specify the size.

  A valid block locator is a non-negative integer (the hash) encoded
  in 32 hex digits. After that it may be followed by a '+' in which
  case the '+' should be followed by the non-negative size (decimal
  encoded). Potentially followed by a '+' and more stuff.

  Source: https://arvados.org/projects/arvados/wiki/Keep_manifest_format
"""
  NOT_A_BLOCK_LOCATOR = [None,None]
  block_locator_tokens = candidate.split('+')

  if len(block_locator_tokens) == 0:
    return NOT_A_BLOCK_LOCATOR

  hash_candidate = block_locator_tokens[0]
  if len(hash_candidate) != BLOCK_HASH_LENGTH:
    return NOT_A_BLOCK_LOCATOR

  block_hash = nonNegativeIntegerFromString(hash_candidate, 16)
  if not block_hash:
    return NOT_A_BLOCK_LOCATOR

  block_size = None
  if len(block_locator_tokens) > 1:
    block_size = nonNegativeIntegerFromString(block_locator_tokens[1], 10)
    if not block_size:
      return NOT_A_BLOCK_LOCATOR

  return block_hash, block_size

def extractUuid(candidate):
  """ Returns a canonical (hash+size) uuid from a valid uuid, or None if candidate is not a valid uuid."""
  match = re.match('([0-9a-fA-F]{32}\+[0-9]+)(\+[^+]+)*$', candidate)
  return match and match.group(1)

def checkUserIsAdmin():
  current_user = arv.users().current().execute()

  if not current_user['is_admin']:
    # TODO(misha): Use a logging framework here
    print ('Warning current user %s (%s - %s) does not have admin access '
           'and will not see much of the data.' %
           (current_user['full_name'],
            current_user['email'],
            current_user['uuid']))
    if args.require_admin_user:
      print 'Exiting, rerun with --no-require-admin-user if you wish to continue.'
      exit(1)

def buildCollectionsList():
  if args.uuid:
    return [args.uuid,]
  else:
    collections_list_response = arv.collections().list(limit=args.max_api_results).execute()

    print ('Returned %d of %d collections.' %
           (len(collections_list_response['items']),
            collections_list_response['items_available']))

    return [item['uuid'] for item in collections_list_response['items']]


def readCollections(collection_uuids):
  for collection_uuid in collection_uuids:
    collection_block_uuids = set()
    collection_response = arv.collections().get(uuid=collection_uuid).execute()
    collection_info = CollectionInfo.get(collection_uuid)
    manifest_lines = collection_response['manifest_text'].split('\n')

    if args.verbose:
      print 'Manifest text for %s:' % collection_uuid
      pprint.pprint(manifest_lines)

    for manifest_line in manifest_lines:
      if manifest_line:
        manifest_tokens = manifest_line.split(' ')
        if args.verbose:
          print 'manifest tokens: ' + pprint.pformat(manifest_tokens)
        stream_name = manifest_tokens[0]

        line_block_uuids = set(filter(None,
                                      [extractUuid(candidate)
                                       for candidate in manifest_tokens[1:]]))
        collection_info.block_uuids.update(line_block_uuids)

        # file_tokens = [token
        #                for token in manifest_tokens[1:]
        #                if extractUuid(token) is None]

        # # Sort file tokens by start position in case they aren't already
        # file_tokens.sort(key=lambda file_token: int(file_token.split(':')[0]))

        # if args.verbose:
        #   print 'line_block_uuids: ' + pprint.pformat(line_block_uuids)
        #   print 'file_tokens: ' + pprint.pformat(file_tokens)


def readLinks():
  link_classes = set()

  for collection_uuid,collection_info in CollectionInfo.all_by_uuid.items():
    collection_links_response = arv.links().list(where={'head_uuid':collection_uuid}).execute()
    link_classes.update([link['link_class'] for link in collection_links_response['items']])
    for link in collection_links_response['items']:
      if link['link_class'] == 'permission':
        collection_info.reader_uuids.add(link['tail_uuid'])
      elif link['link_class'] == 'resources':
        collection_info.persister_uuids.add(link['tail_uuid'])

  print 'Found the following link classes:'
  pprint.pprint(link_classes)

def reportMostPopularCollections():
  most_popular_collections = sorted(
    CollectionInfo.all_by_uuid.values(),
    key=lambda info: len(info.reader_uuids) + 10 * len(info.persister_uuids),
    reverse=True)[:10]

  print 'Most popular Collections:'
  for collection_info in most_popular_collections:
    print collection_info


def buildMaps():
  for collection_uuid,collection_info in CollectionInfo.all_by_uuid.items():
    for block_uuid in collection_info.block_uuids:
      block_to_collections[block_uuid].add(collection_uuid)
      block_to_readers[block_uuid].update(collection_info.reader_uuids)
      block_to_persisters[block_uuid].update(collection_info.persister_uuids)
    for reader_uuid in collection_info.reader_uuids:
      reader_to_collections[reader_uuid].add(collection_uuid)
      reader_to_blocks[reader_uuid].update(collection_info.block_uuids)
    for persister_uuid in collection_info.persister_uuids:
      persister_to_collections[persister_uuid].add(collection_uuid)
      persister_to_blocks[persister_uuid].update(collection_info.block_uuids)


def itemsByValueLength(original):
  return sorted(original.items(),
                key=lambda item:len(item[1]),
                reverse=True)


def reportBusiestUsers():
  busiest_readers = itemsByValueLength(reader_to_collections)
  print 'The busiest readers are:'
  for reader,collections in busiest_readers:
    print '%s reading %d collections.' % (reader, len(collections))
  busiest_persisters = itemsByValueLength(persister_to_collections)
  print 'The busiest persisters are:'
  for persister,collections in busiest_persisters:
    print '%s reading %d collections.' % (persister, len(collections))


def reportUserDiskUsage():
  for user, blocks in reader_to_blocks.items():
    user_to_usage[user][UNWEIGHTED_READ_SIZE_COL] = sum(map(
        byteSizeFromValidUuid,
        blocks))
    user_to_usage[user][WEIGHTED_READ_SIZE_COL] = sum(map(
        lambda block_uuid:(float(byteSizeFromValidUuid(block_uuid))/
                                 len(block_to_readers[block_uuid])),
        blocks))
  for user, blocks in persister_to_blocks.items():
    user_to_usage[user][UNWEIGHTED_PERSIST_SIZE_COL] = sum(map(
        byteSizeFromValidUuid,
        blocks))
    user_to_usage[user][WEIGHTED_PERSIST_SIZE_COL] = sum(map(
        lambda block_uuid:(float(byteSizeFromValidUuid(block_uuid))/
                                 len(block_to_persisters[block_uuid])),
        blocks))
  print ('user: unweighted readable block size, weighted readable block size, '
         'unweighted persisted block size, weighted persisted block size:')
  for user, usage in user_to_usage.items():
    print ('%s: %s %s %s %s' %
           (user,
            fileSizeFormat(usage[UNWEIGHTED_READ_SIZE_COL]),
            fileSizeFormat(usage[WEIGHTED_READ_SIZE_COL]),
            fileSizeFormat(usage[UNWEIGHTED_PERSIST_SIZE_COL]),
            fileSizeFormat(usage[WEIGHTED_PERSIST_SIZE_COL])))



# This is the main flow here

parser = argparse.ArgumentParser(description='Report on keep disks.')
parser.add_argument('-m',
                    '--max-api-results',
                    type=int,
                    default=5000,
                    help=('The max results to get at once.'))
parser.add_argument('-v',
                    '--verbose',
                    help='increase output verbosity',
                    action='store_true')
parser.add_argument('-u',
                    '--uuid',
                    help='uuid of specific collection to process')
parser.add_argument('--require-admin-user',
                    action='store_true',
                    help='Fail if the user is not an admin [default]')
parser.add_argument('--no-require-admin-user',
                    dest='require_admin_user',
                    action='store_false',
                    help='Allow users without admin permissions with only a warning.')
args = parser.parse_args()

checkUserIsAdmin()

print 'Building Collection List'
collection_uuids = filter(None, [extractUuid(candidate)
                                 for candidate in buildCollectionsList()])

print 'Reading Collections'
readCollections(collection_uuids)

if args.verbose:
  pprint.pprint(CollectionInfo.all_by_uuid)

print 'Reading Links'
readLinks()

reportMostPopularCollections()

# These maps all map from uuids to a set of uuids
# The sets all contain collection uuids.
block_to_collections = defaultdict(set)  # keep blocks
reader_to_collections = defaultdict(set)  # collection(s) for which the user has read access
persister_to_collections = defaultdict(set)  # collection(s) which the user has persisted
block_to_readers = defaultdict(set)
block_to_persisters = defaultdict(set)
reader_to_blocks = defaultdict(set)
persister_to_blocks = defaultdict(set)

print 'Building Maps'
buildMaps()

reportBusiestUsers()

UNWEIGHTED_READ_SIZE_COL = 0
WEIGHTED_READ_SIZE_COL = 1
UNWEIGHTED_PERSIST_SIZE_COL = 2
WEIGHTED_PERSIST_SIZE_COL = 3
NUM_COLS = 4
user_to_usage = defaultdict(lambda : [0,]*NUM_COLS)

reportUserDiskUsage()
