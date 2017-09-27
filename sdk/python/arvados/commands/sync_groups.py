# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse
import arvados
import csv
import logging
import os
import sys

from apiclient import errors as apiclient_errors
from arvados._version import __version__

import arvados.commands._util as arv_cmd

api_client = None

GROUP_TAG = 'remote_group'

opts = argparse.ArgumentParser(add_help=False)

opts.add_argument('--version', action='version',
                    version="%s %s" % (sys.argv[0], __version__),
                    help='Print version and exit.')
opts.add_argument('--verbose', action='store_true', default=False,
                  help="""
Log informational messages. By default is deactivated.
""")
opts.add_argument('path', metavar='PATH', type=str, 
                    help="""
Local file path containing a CSV-like format.
""")

_user_id = opts.add_mutually_exclusive_group()
_user_id.add_argument('--user-email', action='store_true', default=True,
                       help="""
Identify users by their email addresses instead of user names.
This is the default.
""")
_user_id.add_argument('--user-name', action='store_false', dest='user_email',
                      help="""
Identify users by their name instead of email addresses.
""")

arg_parser = argparse.ArgumentParser(
    description='Synchronize group memberships from a CSV file.',
    parents=[opts, arv_cmd.retry_opt])

def parse_arguments(arguments):
    args = arg_parser.parse_args(arguments)
    if args.path is None or args.path == '':
        arg_parser.error("Please provide a path to an input file.")
    elif not os.path.exists(args.path):
        arg_parser.error("File not found: '%s'" % args.path)
    elif not os.path.isfile(args.path):
        arg_parser.error("Path provided is not a file: '%s'" % args.path)
    return args

def main(arguments=None, stdout=sys.stdout, stderr=sys.stderr):
    global api_client

    args = parse_arguments(arguments)
    logger = logging.getLogger('arvados.arv_sync_groups')

    if api_client is None:
        api_client = arvados.api('v1')

    # How are users going to be identified on the input file?
    if args.user_email:
        user_id = 'email'
    else:
        user_id = 'username'
    
    if args.verbose:
        logger.setLevel(logging.INFO)
        
    logger.info("Group sync starting. Using '%s' as users id" % user_id)
    
    # Get the complete user list to minimize API Server requests
    all_users = {}
    userid_to_uuid = {} # Index by user_id (email/username)
    for u in arvados.util.list_all(api_client.users().list, args.retries):
        all_users[u['uuid']] = u
        userid_to_uuid[u[user_id]] = u['uuid']
    logger.info('Found %d users' % len(all_users))

    # Request all UUIDs for groups tagged as remote
    remote_group_uuids = set()
    for link in arvados.util.list_all(
                            api_client.links().list, 
                            args.retries,
                            filters=[['link_class', '=', 'tag'],
                                     ['name', '=', GROUP_TAG],
                                     ['head_kind', '=', 'arvados#group']]):
        remote_group_uuids.add(link['head_uuid'])
    # Get remote groups and their members
    remote_groups = {}
    group_name_to_uuid = {} # Index by group name
    for group in arvados.util.list_all(
                            api_client.groups().list,
                            args.retries,
                            filters=[['uuid', 'in', list(remote_group_uuids)]]):
        member_links = arvados.util.list_all(
                            api_client.links().list,
                            args.retries,
                            filters=[['link_class', '=', 'permission'],
                                      ['name', '=', 'can_read'],
                                      ['tail_uuid', '=', group['uuid']],
                                      ['head_kind', '=', 'arvados#user']])
        # Build a list of user_ids (email/username) belonging to this group
        members = set([all_users[link['head_uuid']][user_id] 
                       for link in member_links])
        remote_groups[group['uuid']] = {'object': group,
                                        'previous_members': members,
                                        'current_members': set()}
        # FIXME: There's an index (group_name, group.owner_uuid), should we
        # ask for our own groups tagged as remote? (with own being 'system'?)
        group_name_to_uuid[group['name']] = group['uuid']
    logger.info('Found %d remote groups' % len(remote_groups))
    
    groups_created = 0
    members_added = 0
    members_removed = 0
    with open(args.path, 'rb') as f:
        reader = csv.reader(f)
        try:
            for group, user in reader:
                group = group.strip()
                user = user.strip()
                if not user in userid_to_uuid:
                    # User not present on the system, skip.
                    logger.warning("There's no user with %s '%s' on the system"
                                   ", skipping." % (user_id, user))
                    continue
                if not group in group_name_to_uuid:
                    # Group doesn't exist, create and tag it before continuing
                    g = api_client.groups().create(body={
                        'name': group}).execute(num_retries=args.retries)
                    api_client.links().create(body={
                        'link_class': 'tag',
                        'name': GROUP_TAG,
                        'head_uuid': g['uuid'],
                    }).execute(num_retries=args.retries)
                    # Update cached group data
                    group_name_to_uuid[g['name']] = g['uuid']
                    remote_groups[g['uuid']] = {'object': g,
                                                'previous_members': set(),
                                                'current_members': set()}
                    groups_created += 1
                # Both group & user exist, check if user is a member
                g_uuid = group_name_to_uuid[group]
                if not (user in remote_groups[g_uuid]['previous_members'] or
                        user in remote_groups[g_uuid]['current_members']):
                    # User wasn't a member, but should.
                    api_client.links().create(body={
                        'link_class': 'permission',
                        'name': 'can_read',
                        'tail_uuid': g_uuid,
                        'head_uuid': userid_to_uuid[user],
                    }).execute(num_retries=args.retries)
                    members_added += 1
                remote_groups[g_uuid]['current_members'].add(user)
        except (ValueError, csv.Error) as e:
            logger.warning('Error on line %d: %s' % (reader.line_num, e))
    # Remove previous members not listed on this run
    for group_uuid in remote_groups:
        previous = remote_groups[group_uuid]['previous_members']
        current = remote_groups[group_uuid]['current_members']
        evicted = previous - current
        if len(evicted) > 0:
            logger.info("Removing %d users from group '%s'" % (
                len(evicted), remote_groups[group_uuid]['object']['name']))
        for evicted_user in evicted:
            links = arvados.util.list_all(
                api_client.links().list,
                args.retries,
                filters=[['link_class', '=', 'permission'],
                         ['name', '=', 'can_read'],
                         ['tail_uuid', '=', group_uuid],
                         ['head_uuid', '=', userid_to_uuid[evicted_user]]])
            for l in links:
                api_client.links().delete(
                    uuid=l['uuid']).execute(num_retries=args.retries)
            members_removed += 1
    logger.info("Groups created: %d, members added: %s, members removed: %d" % \
                (groups_created, members_added, members_removed))

if __name__ == '__main__':
    main()
