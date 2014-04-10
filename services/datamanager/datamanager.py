#! /usr/bin/env python

import arvados

import argparse
import cgi
import logging
import math
import pprint
import re
import threading
import urllib2

from BaseHTTPServer import HTTPServer, BaseHTTPRequestHandler
from collections import defaultdict, Counter
from functools import partial
from operator import itemgetter
from SocketServer import ThreadingMixIn

arv = arvados.api('v1')

# Adapted from http://stackoverflow.com/questions/4180980/formatting-data-quantity-capacity-as-string
byteunits = ('B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB')
def fileSizeFormat(value):
  exponent = 0 if value == 0 else int(math.log(value, 1024))
  return "%7.2f %-3s" % (float(value) / pow(1024, exponent),
                         byteunits[exponent])

def byteSizeFromValidUuid(valid_uuid):
  return int(valid_uuid.split('+')[1])

class maxdict(dict):
  """A dictionary that holds the largest value entered for each key."""
  def addValue(self, key, value):
    dict.__setitem__(self, key, max(dict.get(self, key), value))
  def addValues(self, kv_pairs):
    for key,value in kv_pairs:
      self.addValue(key, value)
  def addDict(self, d):
    self.addValues(d.items())

class CollectionInfo:
  DEFAULT_PERSISTER_REPLICATION_LEVEL=2
  all_by_uuid = {}

  def __init__(self, uuid):
    if CollectionInfo.all_by_uuid.has_key(uuid):
      raise ValueError('Collection for uuid "%s" already exists.' % uuid)
    self.uuid = uuid
    self.block_uuids = set()  # uuids of keep blocks in this collection
    self.reader_uuids = set()  # uuids of users who can read this collection
    self.persister_uuids = set()  # uuids of users who want this collection saved
    # map from user uuid to replication level they desire
    self.persister_replication = maxdict()

    # The whole api response in case we need anything else later.
    self.api_response = []
    CollectionInfo.all_by_uuid[uuid] = self

  def byteSize(self):
    return sum(map(byteSizeFromValidUuid, self.block_uuids))

  def __str__(self):
    return ('CollectionInfo uuid: %s\n'
            '               %d block(s) containing %s\n'
            '               reader_uuids: %s\n'
            '               persister_replication: %s' %
            (self.uuid,
             len(self.block_uuids),
             fileSizeFormat(self.byteSize()),
             pprint.pformat(self.reader_uuids, indent = 15),
             pprint.pformat(self.persister_replication, indent = 15)))

  @staticmethod
  def get(uuid):
    if not CollectionInfo.all_by_uuid.has_key(uuid):
      CollectionInfo(uuid)
    return CollectionInfo.all_by_uuid[uuid]
  

def extractUuid(candidate):
  """ Returns a canonical (hash+size) uuid from a valid uuid, or None if candidate is not a valid uuid."""
  match = re.match('([0-9a-fA-F]{32}\+[0-9]+)(\+[^+]+)*$', candidate)
  return match and match.group(1)

def checkUserIsAdmin():
  current_user = arv.users().current().execute()

  if not current_user['is_admin']:
    log.warning('Current user %s (%s - %s) does not have '
                'admin access and will not see much of the data.',
                current_user['full_name'],
                current_user['email'],
                current_user['uuid'])
    if args.require_admin_user:
      log.critical('Exiting, rerun with --no-require-admin-user '
                   'if you wish to continue.')
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
    collection_info.api_response = collection_response
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
        replication_level = link['properties'].get(
          'replication',
          CollectionInfo.DEFAULT_PERSISTER_REPLICATION_LEVEL)
        collection_info.persister_replication.addValue(
          link['tail_uuid'],
          replication_level)
        collection_info.persister_uuids.add(link['tail_uuid'])

  print 'Found the following link classes:'
  pprint.pprint(link_classes)

def reportMostPopularCollections():
  most_popular_collections = sorted(
    CollectionInfo.all_by_uuid.values(),
    key=lambda info: len(info.reader_uuids) + 10 * len(info.persister_replication),
    reverse=True)[:10]

  print 'Most popular Collections:'
  for collection_info in most_popular_collections:
    print collection_info


def buildMaps():
  for collection_uuid,collection_info in CollectionInfo.all_by_uuid.items():
    # Add the block holding the manifest itself for all calculations
    block_uuids = collection_info.block_uuids.union([collection_uuid,])
    for block_uuid in block_uuids:
      block_to_collections[block_uuid].add(collection_uuid)
      block_to_readers[block_uuid].update(collection_info.reader_uuids)
      block_to_persisters[block_uuid].update(collection_info.persister_uuids)
      block_to_persister_replication[block_uuid].addDict(
        collection_info.persister_replication)
    for reader_uuid in collection_info.reader_uuids:
      reader_to_collections[reader_uuid].add(collection_uuid)
      reader_to_blocks[reader_uuid].update(block_uuids)
    for persister_uuid in collection_info.persister_uuids:
      persister_to_collections[persister_uuid].add(collection_uuid)
      persister_to_blocks[persister_uuid].update(block_uuids)


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


def blockDiskUsage(block_uuid):
  """Returns the disk usage of a block given its uuid.

  Will return 0 before reading the contents of the keep servers.
  """
  return byteSizeFromValidUuid(block_uuid) * block_to_replication[block_uuid]

def blockPersistedUsage(user_uuid, block_uuid):
  return (byteSizeFromValidUuid(block_uuid) *
          block_to_persister_replication[block_uuid].get(user_uuid, 0))

memo_computeWeightedReplicationCosts = {}
def computeWeightedReplicationCosts(replication_levels):
  """Computes the relative cost of varied replication levels.

  replication_levels: a tuple of integers representing the desired
  replication level. If n users want a replication level of x then x
  should appear n times in replication_levels.

  Returns a dictionary from replication level to cost.

  The basic thinking is that the cost of replicating at level x should
  be shared by everyone who wants replication of level x or higher.

  For example, if we have two users who want 1 copy, one user who
  wants 3 copies and two users who want 6 copies:
  the input would be [1, 1, 3, 6, 6] (or any permutation)

  The cost of the first copy is shared by all 5 users, so they each
  pay 1 copy / 5 users = 0.2.
  The cost of the second and third copies shared by 3 users, so they
  each pay 2 copies / 3 users = 0.67 (plus the above costs)
  The cost of the fourth, fifth and sixth copies is shared by two
  users, so they each pay 3 copies / 2 users = 1.5 (plus the above costs)

  Here are some sample other examples:
  computeWeightedReplicationCosts([1,]) -> {1:1.0}
  computeWeightedReplicationCosts([2,]) -> {2:2.0}
  computeWeightedReplicationCosts([1,1]) -> {1:0.5}
  computeWeightedReplicationCosts([2,2]) -> {1:1.0}
  computeWeightedReplicationCosts([1,2]) -> {1:0.5,2:1.5}
  computeWeightedReplicationCosts([1,3]) -> {1:0.5,2:2.5}
  computeWeightedReplicationCosts([1,3,6,6,10]) -> {1:0.2,3:0.7,6:1.7,10:5.7}
  """
  replication_level_counts = sorted(Counter(replication_levels).items())

  memo_key = str(replication_level_counts)

  if not memo_key in memo_computeWeightedReplicationCosts:
    last_level = 0
    current_cost = 0
    total_interested = float(sum(map(itemgetter(1), replication_level_counts)))
    cost_for_level = {}
    for replication_level, count in replication_level_counts:
      copies_added = replication_level - last_level
      # compute marginal cost from last level and add it to the last cost
      current_cost += copies_added / total_interested
      cost_for_level[replication_level] = current_cost
      # update invariants
      last_level = replication_level
      total_interested -= count
    memo_computeWeightedReplicationCosts[memo_key] = cost_for_level

  return memo_computeWeightedReplicationCosts[memo_key]

def blockPersistedWeightedUsage(user_uuid, block_uuid):
  persister_replication_for_block = block_to_persister_replication[block_uuid]
  user_replication = persister_replication_for_block[user_uuid]
  return (
    byteSizeFromValidUuid(block_uuid) *
    computeWeightedReplicationCosts(
      persister_replication_for_block.values())[user_replication])

def reportUserDiskUsage():
  for user, blocks in reader_to_blocks.items():
    user_to_usage[user][UNWEIGHTED_READ_SIZE_COL] = sum(map(
        blockDiskUsage,
        blocks))
    user_to_usage[user][WEIGHTED_READ_SIZE_COL] = sum(map(
        lambda block_uuid:(float(blockDiskUsage(block_uuid))/
                                 len(block_to_readers[block_uuid])),
        blocks))
  for user, blocks in persister_to_blocks.items():
    user_to_usage[user][UNWEIGHTED_PERSIST_SIZE_COL] = sum(map(
        partial(blockPersistedUsage, user),
        blocks))
    user_to_usage[user][WEIGHTED_PERSIST_SIZE_COL] = sum(map(
        partial(blockPersistedWeightedUsage, user),
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


def getKeepServers():
  response = arv.keep_disks().list().execute()
  return [[keep_server['service_host'], keep_server['service_port']]
          for keep_server in response['items']]


def getKeepBlocks(keep_servers):
  blocks = []
  for host,port in keep_servers:
    response = urllib2.urlopen('http://%s:%d/index' % (host, port))
    blocks.append([line.split(' ')
                   for line in response.read().split('\n')
                   if line])
  return blocks


def computeReplication(keep_blocks):
  for server_blocks in keep_blocks:
    for block_uuid, _ in server_blocks:
      block_to_replication[block_uuid] += 1


# This is the main flow here


args = None

log = logging.getLogger('arvados.services.datamanager')
stderr_handler = logging.StreamHandler()
log.setLevel(logging.INFO)
stderr_handler.setFormatter(
  logging.Formatter('%(asctime)-15s %(levelname)-8s %(message)s'))
log.addHandler(stderr_handler)

# Global Data - don't try this at home
collection_uuids = []

# These maps all map from uuids to a set of uuids
block_to_collections = defaultdict(set)  # keep blocks
reader_to_collections = defaultdict(set)  # collection(s) for which the user has read access
persister_to_collections = defaultdict(set)  # collection(s) which the user has persisted
block_to_readers = defaultdict(set)
block_to_persisters = defaultdict(set)
block_to_persister_replication = defaultdict(maxdict)
reader_to_blocks = defaultdict(set)
persister_to_blocks = defaultdict(set)

UNWEIGHTED_READ_SIZE_COL = 0
WEIGHTED_READ_SIZE_COL = 1
UNWEIGHTED_PERSIST_SIZE_COL = 2
WEIGHTED_PERSIST_SIZE_COL = 3
NUM_COLS = 4
user_to_usage = defaultdict(lambda : [0,]*NUM_COLS)

keep_servers = []
keep_blocks = []
block_to_replication = defaultdict(lambda: 0)

all_data_loaded = False

def loadAllData():
  checkUserIsAdmin()

  log.info('Building Collection List')
  global collection_uuids
  collection_uuids = filter(None, [extractUuid(candidate)
                                   for candidate in buildCollectionsList()])

  log.info('Reading Collections')
  readCollections(collection_uuids)

  if args.verbose:
    pprint.pprint(CollectionInfo.all_by_uuid)

  log.info('Reading Links')
  readLinks()

  reportMostPopularCollections()

  log.info('Building Maps')
  buildMaps()

  reportBusiestUsers()

  log.info('Getting Keep Servers')
  global keep_servers
  keep_servers = getKeepServers()

  print keep_servers

  log.info('Getting Blocks from each Keep Server.')
  global keep_blocks
  keep_blocks = getKeepBlocks(keep_servers)

  computeReplication(keep_blocks)

  log.info('average replication level is %f', (float(sum(block_to_replication.values())) / len(block_to_replication)))

  reportUserDiskUsage()

  global all_data_loaded
  all_data_loaded = True

class DataManagerHandler(BaseHTTPRequestHandler):
  USER_PATH = 'user'
  COLLECTION_PATH = 'collection'
  BLOCK_PATH = 'block'

  def userLink(self, uuid):
    return ('<A HREF="/%(path)s/%(uuid)s">%(uuid)s</A>' %
            {'uuid': uuid,
             'path': DataManagerHandler.USER_PATH})

  def collectionLink(self, uuid):
    return ('<A HREF="/%(path)s/%(uuid)s">%(uuid)s</A>' %
            {'uuid': uuid,
             'path': DataManagerHandler.COLLECTION_PATH})

  def blockLink(self, uuid):
    return ('<A HREF="/%(path)s/%(uuid)s">%(uuid)s</A>' %
            {'uuid': uuid,
             'path': DataManagerHandler.BLOCK_PATH})

  def writeTop(self, title):
    self.wfile.write('<HTML><HEAD><TITLE>%s</TITLE></HEAD>\n<BODY>' % title)
    
  def writeBottom(self):
    self.wfile.write('</BODY></HTML>\n')
    
  def writeHomePage(self):
    self.send_response(200)
    self.end_headers()
    self.writeTop('Home')
    self.wfile.write('<TABLE>')
    self.wfile.write('<TR><TH>user'
                     '<TH>unweighted readable block size'
                     '<TH>weighted readable block size'
                     '<TH>unweighted persisted block size'
                     '<TH>weighted persisted block size</TR>\n')
    for user, usage in user_to_usage.items():
      self.wfile.write('<TR><TD>%s<TD>%s<TD>%s<TD>%s<TD>%s</TR>\n' %
                       (self.userLink(user),
                        fileSizeFormat(usage[UNWEIGHTED_READ_SIZE_COL]),
                        fileSizeFormat(usage[WEIGHTED_READ_SIZE_COL]),
                        fileSizeFormat(usage[UNWEIGHTED_PERSIST_SIZE_COL]),
                        fileSizeFormat(usage[WEIGHTED_PERSIST_SIZE_COL])))
    self.wfile.write('</TABLE>\n')
    self.writeBottom()

  def userExists(self, uuid):
    # Currently this will return false for a user who exists but
    # doesn't appear on any manifests.
    # TODO(misha): Figure out if we need to fix this.
    return user_to_usage.has_key(uuid)

  def writeUserPage(self, uuid):
    if not self.userExists(uuid):
      self.send_error(404,
                      'User (%s) Not Found.' % cgi.escape(uuid, quote=False))
    else:
      # Here we assume that since a user exists, they don't need to be
      # html escaped.
      self.send_response(200)
      self.end_headers()
      self.writeTop('User %s' % uuid)
      self.wfile.write('<TABLE>')
      self.wfile.write('<TR><TH>user'
                       '<TH>unweighted readable block size'
                       '<TH>weighted readable block size'
                       '<TH>unweighted persisted block size'
                       '<TH>weighted persisted block size</TR>\n')
      usage = user_to_usage[uuid]
      self.wfile.write('<TR><TD>%s<TD>%s<TD>%s<TD>%s<TD>%s</TR>\n' %
                       (self.userLink(uuid),
                        fileSizeFormat(usage[UNWEIGHTED_READ_SIZE_COL]),
                        fileSizeFormat(usage[WEIGHTED_READ_SIZE_COL]),
                        fileSizeFormat(usage[UNWEIGHTED_PERSIST_SIZE_COL]),
                        fileSizeFormat(usage[WEIGHTED_PERSIST_SIZE_COL])))
      self.wfile.write('</TABLE>\n')
      self.wfile.write('<P>Persisting Collections: %s\n' %
                       ', '.join(map(self.collectionLink,
                                     persister_to_collections[uuid])))
      self.wfile.write('<P>Reading Collections: %s\n' %
                       ', '.join(map(self.collectionLink,
                                     reader_to_collections[uuid])))
      self.writeBottom()

  def collectionExists(self, uuid):
    return CollectionInfo.all_by_uuid.has_key(uuid)

  def writeCollectionPage(self, uuid):
    if not self.collectionExists(uuid):
      self.send_error(404,
                      'Collection (%s) Not Found.' % cgi.escape(uuid, quote=False))
    else:
      collection = CollectionInfo.get(uuid)
      # Here we assume that since a collection exists, its id doesn't
      # need to be html escaped.
      self.send_response(200)
      self.end_headers()
      self.writeTop('Collection %s' % uuid)
      self.wfile.write('<H1>Collection %s</H1>\n' % uuid)
      self.wfile.write('<P>Total size %s (not factoring in replication).\n' %
                       fileSizeFormat(collection.byteSize()))
      self.wfile.write('<P>Readers: %s\n' %
                       ', '.join(map(self.userLink, collection.reader_uuids)))

      if len(collection.persister_replication) == 0:
        self.wfile.write('<P>No persisters\n')
      else:
        replication_to_users = defaultdict(set)
        for user,replication in collection.persister_replication.items():
          replication_to_users[replication].add(user)
        replication_levels = sorted(replication_to_users.keys())

        self.wfile.write('<P>%d persisters in %d replication level(s) maxing '
                         'out at %dx replication:\n' %
                         (len(collection.persister_replication),
                          len(replication_levels),
                          replication_levels[-1]))

        # TODO(misha): This code is used twice, let's move it to a method.
        self.wfile.write('<TABLE><TR><TH>%s</TR>\n' %
                         '<TH>'.join(['Replication Level ' + str(x)
                                      for x in replication_levels]))
        self.wfile.write('<TR>\n')
        for replication_level in replication_levels:
          users = replication_to_users[replication_level]
          self.wfile.write('<TD valign="top">%s\n' % '<BR>\n'.join(
              map(self.userLink, users)))
        self.wfile.write('</TR></TABLE>\n')

      replication_to_blocks = defaultdict(set)
      for block in collection.block_uuids:
        replication_to_blocks[block_to_replication[block]].add(block)
      replication_levels = sorted(replication_to_blocks.keys())
      self.wfile.write('<P>%d blocks in %d replication level(s):\n' %
                       (len(collection.block_uuids), len(replication_levels)))
      self.wfile.write('<TABLE><TR><TH>%s</TR>\n' %
                       '<TH>'.join(['Replication Level ' + str(x)
                                    for x in replication_levels]))
      self.wfile.write('<TR>\n')
      for replication_level in replication_levels:
        blocks = replication_to_blocks[replication_level]
        self.wfile.write('<TD valign="top">%s\n' % '<BR>\n'.join(blocks))
      self.wfile.write('</TR></TABLE>\n')
      

  def do_GET(self):
    if not all_data_loaded:
      self.send_error(503,
                      'Sorry, but I am still loading all the data I need.')
    else:
      # Removing leading '/' and process request path
      split_path = self.path[1:].split('/')
      request_type = split_path[0]
      log.debug('path (%s) split as %s with request_type %s' % (self.path,
                                                                split_path,
                                                                request_type))
      if request_type == '':
        self.writeHomePage()
      elif request_type == DataManagerHandler.USER_PATH:
        self.writeUserPage(split_path[1])
      elif request_type == DataManagerHandler.COLLECTION_PATH:
        self.writeCollectionPage(split_path[1])
      else:
        self.send_error(404, 'Unrecognized request path.')
    return

class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
  """Handle requests in a separate thread."""

if __name__ == '__main__':
  parser = argparse.ArgumentParser(description='Report on keep disks.')
  parser.add_argument('-m',
                      '--max-api-results',
                      type=int,
                      default=5000,
                      help=('The max results to get at once.'))
  parser.add_argument('-p',
                      '--port',
                      type=int,
                      default=9090,
                      help=('The port number to serve on. 0 means no server.'))
  parser.add_argument('-v',
                      '--verbose',
                      help='increase output verbosity',
                      action='store_true')
  parser.add_argument('-u',
                      '--uuid',
                      help='uuid of specific collection to process')
  parser.add_argument('--require-admin-user',
                      action='store_true',
                      default=True,
                      help='Fail if the user is not an admin [default]')
  parser.add_argument('--no-require-admin-user',
                      dest='require_admin_user',
                      action='store_false',
                      help='Allow users without admin permissions with only a warning.')


  args = parser.parse_args()


  if args.port == 0:
    loadAllData()
  else:
    loader = threading.Thread(target = loadAllData, name = 'loader')
    loader.start()

    server = ThreadedHTTPServer(('localhost', args.port), DataManagerHandler)
    server.serve_forever()
