# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from builtins import next
import argparse
import collections
import datetime
import errno
import json
import os
import re
import subprocess
import sys
import tarfile
import tempfile
import shutil
import _strptime

from operator import itemgetter
from stat import *

import arvados
import arvados.util
import arvados.commands._util as arv_cmd
import arvados.commands.put as arv_put
from arvados.collection import CollectionReader
import ciso8601
import logging
import arvados.config

from arvados._version import __version__

logger = logging.getLogger('arvados.keepdocker')
logger.setLevel(logging.DEBUG if arvados.config.get('ARVADOS_DEBUG')
                else logging.INFO)

EARLIEST_DATETIME = datetime.datetime(datetime.MINYEAR, 1, 1, 0, 0, 0)
STAT_CACHE_ERRORS = (IOError, OSError, ValueError)

DockerImage = collections.namedtuple(
    'DockerImage', ['repo', 'tag', 'hash', 'created', 'vsize'])

keepdocker_parser = argparse.ArgumentParser(add_help=False)
keepdocker_parser.add_argument(
    '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
    help='Print version and exit.')
keepdocker_parser.add_argument(
    '-f', '--force', action='store_true', default=False,
    help="Re-upload the image even if it already exists on the server")
keepdocker_parser.add_argument(
    '--force-image-format', action='store_true', default=False,
    help="Proceed even if the image format is not supported by the server")

_group = keepdocker_parser.add_mutually_exclusive_group()
_group.add_argument(
    '--pull', action='store_true', default=False,
    help="Try to pull the latest image from Docker registry")
_group.add_argument(
    '--no-pull', action='store_false', dest='pull',
    help="Use locally installed image only, don't pull image from Docker registry (default)")

keepdocker_parser.add_argument(
    'image', nargs='?',
    help="Docker image to upload: repo, repo:tag, or hash")
keepdocker_parser.add_argument(
    'tag', nargs='?',
    help="Tag of the Docker image to upload (default 'latest'), if image is given as an untagged repo name")

# Combine keepdocker options listed above with run_opts options of arv-put.
# The options inherited from arv-put include --name, --project-uuid,
# --progress/--no-progress/--batch-progress and --resume/--no-resume.
arg_parser = argparse.ArgumentParser(
        description="Upload or list Docker images in Arvados",
        parents=[keepdocker_parser, arv_put.run_opts, arv_cmd.retry_opt])

class DockerError(Exception):
    pass


def popen_docker(cmd, *args, **kwargs):
    manage_stdin = ('stdin' not in kwargs)
    kwargs.setdefault('stdin', subprocess.PIPE)
    kwargs.setdefault('stdout', sys.stderr)
    try:
        docker_proc = subprocess.Popen(['docker.io'] + cmd, *args, **kwargs)
    except OSError:  # No docker.io in $PATH
        docker_proc = subprocess.Popen(['docker'] + cmd, *args, **kwargs)
    if manage_stdin:
        docker_proc.stdin.close()
    return docker_proc

def check_docker(proc, description):
    proc.wait()
    if proc.returncode != 0:
        raise DockerError("docker {} returned status code {}".
                          format(description, proc.returncode))

def docker_image_format(image_hash):
    """Return the registry format ('v1' or 'v2') of the given image."""
    cmd = popen_docker(['inspect', '--format={{.Id}}', image_hash],
                        stdout=subprocess.PIPE)
    try:
        image_id = next(cmd.stdout).decode().strip()
        if image_id.startswith('sha256:'):
            return 'v2'
        elif ':' not in image_id:
            return 'v1'
        else:
            return 'unknown'
    finally:
        check_docker(cmd, "inspect")

def docker_image_compatible(api, image_hash):
    supported = api._rootDesc.get('dockerImageFormats', [])
    if not supported:
        logger.warning("server does not specify supported image formats (see docker_image_formats in server config).")
        return False

    fmt = docker_image_format(image_hash)
    if fmt in supported:
        return True
    else:
        logger.error("image format is {!r} " \
            "but server supports only {!r}".format(fmt, supported))
        return False

def docker_images():
    # Yield a DockerImage tuple for each installed image.
    list_proc = popen_docker(['images', '--no-trunc'], stdout=subprocess.PIPE)
    list_output = iter(list_proc.stdout)
    next(list_output)  # Ignore the header line
    for line in list_output:
        words = line.split()
        size_index = len(words) - 2
        repo, tag, imageid = words[:3]
        ctime = ' '.join(words[3:size_index])
        vsize = ' '.join(words[size_index:])
        yield DockerImage(repo, tag, imageid, ctime, vsize)
    list_proc.stdout.close()
    check_docker(list_proc, "images")

def find_image_hashes(image_search, image_tag=None):
    # Given one argument, search for Docker images with matching hashes,
    # and return their full hashes in a set.
    # Given two arguments, also search for a Docker image with the
    # same repository and tag.  If one is found, return its hash in a
    # set; otherwise, fall back to the one-argument hash search.
    # Returns None if no match is found, or a hash search is ambiguous.
    hash_search = image_search.lower()
    hash_matches = set()
    for image in docker_images():
        if (image.repo == image_search) and (image.tag == image_tag):
            return set([image.hash])
        elif image.hash.startswith(hash_search):
            hash_matches.add(image.hash)
    return hash_matches

def find_one_image_hash(image_search, image_tag=None):
    hashes = find_image_hashes(image_search, image_tag)
    hash_count = len(hashes)
    if hash_count == 1:
        return hashes.pop()
    elif hash_count == 0:
        raise DockerError("no matching image found")
    else:
        raise DockerError("{} images match {}".format(hash_count, image_search))

def stat_cache_name(image_file):
    return getattr(image_file, 'name', image_file) + '.stat'

def pull_image(image_name, image_tag):
    check_docker(popen_docker(['pull', '{}:{}'.format(image_name, image_tag)]),
                 "pull")

def save_image(image_hash, image_file):
    # Save the specified Docker image to image_file, then try to save its
    # stats so we can try to resume after interruption.
    check_docker(popen_docker(['save', image_hash], stdout=image_file),
                 "save")
    image_file.flush()
    try:
        with open(stat_cache_name(image_file), 'w') as statfile:
            json.dump(tuple(os.fstat(image_file.fileno())), statfile)
    except STAT_CACHE_ERRORS:
        pass  # We won't resume from this cache.  No big deal.

def prep_image_file(filename):
    # Return a file object ready to save a Docker image,
    # and a boolean indicating whether or not we need to actually save the
    # image (False if a cached save is available).
    cache_dir = arv_cmd.make_home_conf_dir(
        os.path.join('.cache', 'arvados', 'docker'), 0o700)
    if cache_dir is None:
        image_file = tempfile.NamedTemporaryFile(suffix='.tar')
        need_save = True
    else:
        file_path = os.path.join(cache_dir, filename)
        try:
            with open(stat_cache_name(file_path)) as statfile:
                prev_stat = json.load(statfile)
            now_stat = os.stat(file_path)
            need_save = any(prev_stat[field] != now_stat[field]
                            for field in [ST_MTIME, ST_SIZE])
        except STAT_CACHE_ERRORS + (AttributeError, IndexError):
            need_save = True  # We couldn't compare against old stats
        image_file = open(file_path, 'w+b' if need_save else 'rb')
    return image_file, need_save

def make_link(api_client, num_retries, link_class, link_name, **link_attrs):
    link_attrs.update({'link_class': link_class, 'name': link_name})
    return api_client.links().create(body=link_attrs).execute(
        num_retries=num_retries)

def docker_link_sort_key(link):
    """Build a sort key to find the latest available Docker image.

    To find one source collection for a Docker image referenced by
    name or image id, the API server looks for a link with the most
    recent `image_timestamp` property; then the most recent
    `created_at` timestamp.  This method generates a sort key for
    Docker metadata links to sort them from least to most preferred.
    """
    try:
        image_timestamp = ciso8601.parse_datetime_unaware(
            link['properties']['image_timestamp'])
    except (KeyError, ValueError):
        image_timestamp = EARLIEST_DATETIME
    return (image_timestamp,
            ciso8601.parse_datetime_unaware(link['created_at']))

def _get_docker_links(api_client, num_retries, **kwargs):
    links = arvados.util.list_all(api_client.links().list,
                                  num_retries, **kwargs)
    for link in links:
        link['_sort_key'] = docker_link_sort_key(link)
    links.sort(key=itemgetter('_sort_key'), reverse=True)
    return links

def _new_image_listing(link, dockerhash, repo='<none>', tag='<none>'):
    timestamp_index = 1 if (link['_sort_key'][0] is EARLIEST_DATETIME) else 0
    return {
        '_sort_key': link['_sort_key'],
        'timestamp': link['_sort_key'][timestamp_index],
        'collection': link['head_uuid'],
        'dockerhash': dockerhash,
        'repo': repo,
        'tag': tag,
        }

def list_images_in_arv(api_client, num_retries, image_name=None, image_tag=None):
    """List all Docker images known to the api_client with image_name and
    image_tag.  If no image_name is given, defaults to listing all
    Docker images.

    Returns a list of tuples representing matching Docker images,
    sorted in preference order (i.e. the first collection in the list
    is the one that the API server would use). Each tuple is a
    (collection_uuid, collection_info) pair, where collection_info is
    a dict with fields "dockerhash", "repo", "tag", and "timestamp".

    """
    search_filters = []
    repo_links = None
    hash_links = None
    if image_name:
        # Find images with the name the user specified.
        search_links = _get_docker_links(
            api_client, num_retries,
            filters=[['link_class', '=', 'docker_image_repo+tag'],
                     ['name', '=',
                      '{}:{}'.format(image_name, image_tag or 'latest')]])
        if search_links:
            repo_links = search_links
        else:
            # Fall back to finding images with the specified image hash.
            search_links = _get_docker_links(
                api_client, num_retries,
                filters=[['link_class', '=', 'docker_image_hash'],
                         ['name', 'ilike', image_name + '%']])
            hash_links = search_links
        # Only list information about images that were found in the search.
        search_filters.append(['head_uuid', 'in',
                               [link['head_uuid'] for link in search_links]])

    # It should be reasonable to expect that each collection only has one
    # image hash (though there may be many links specifying this).  Find
    # the API server's most preferred image hash link for each collection.
    if hash_links is None:
        hash_links = _get_docker_links(
            api_client, num_retries,
            filters=search_filters + [['link_class', '=', 'docker_image_hash']])
    hash_link_map = {link['head_uuid']: link for link in reversed(hash_links)}

    # Each collection may have more than one name (though again, one name
    # may be specified more than once).  Build an image listing from name
    # tags, sorted by API server preference.
    if repo_links is None:
        repo_links = _get_docker_links(
            api_client, num_retries,
            filters=search_filters + [['link_class', '=',
                                       'docker_image_repo+tag']])
    seen_image_names = collections.defaultdict(set)
    images = []
    for link in repo_links:
        collection_uuid = link['head_uuid']
        if link['name'] in seen_image_names[collection_uuid]:
            continue
        seen_image_names[collection_uuid].add(link['name'])
        try:
            dockerhash = hash_link_map[collection_uuid]['name']
        except KeyError:
            dockerhash = '<unknown>'
        name_parts = link['name'].split(':', 1)
        images.append(_new_image_listing(link, dockerhash, *name_parts))

    # Find any image hash links that did not have a corresponding name link,
    # and add image listings for them, retaining the API server preference
    # sorting.
    images_start_size = len(images)
    for collection_uuid, link in hash_link_map.items():
        if not seen_image_names[collection_uuid]:
            images.append(_new_image_listing(link, link['name']))
    if len(images) > images_start_size:
        images.sort(key=itemgetter('_sort_key'), reverse=True)

    # Remove any image listings that refer to unknown collections.
    existing_coll_uuids = {coll['uuid'] for coll in arvados.util.list_all(
            api_client.collections().list, num_retries,
            filters=[['uuid', 'in', [im['collection'] for im in images]]],
            select=['uuid'])}
    return [(image['collection'], image) for image in images
            if image['collection'] in existing_coll_uuids]

def items_owned_by(owner_uuid, arv_items):
    return (item for item in arv_items if item['owner_uuid'] == owner_uuid)

def _uuid2pdh(api, uuid):
    return api.collections().list(
        filters=[['uuid', '=', uuid]],
        select=['portable_data_hash'],
    ).execute()['items'][0]['portable_data_hash']

def main(arguments=None, stdout=sys.stdout):
    args = arg_parser.parse_args(arguments)
    api = arvados.api('v1')

    if args.image is None or args.image == 'images':
        fmt = "{:30}  {:10}  {:12}  {:29}  {:20}\n"
        stdout.write(fmt.format("REPOSITORY", "TAG", "IMAGE ID", "COLLECTION", "CREATED"))
        try:
            for i, j in list_images_in_arv(api, args.retries):
                stdout.write(fmt.format(j["repo"], j["tag"], j["dockerhash"][0:12], i, j["timestamp"].strftime("%c")))
        except IOError as e:
            if e.errno == errno.EPIPE:
                pass
            else:
                raise
        sys.exit(0)

    if re.search(r':\w[-.\w]{0,127}$', args.image):
        # image ends with :valid-tag
        if args.tag is not None:
            logger.error(
                "image %r already includes a tag, cannot add tag argument %r",
                args.image, args.tag)
            sys.exit(1)
        # rsplit() accommodates "myrepo.example:8888/repo/image:tag"
        args.image, args.tag = args.image.rsplit(':', 1)
    elif args.tag is None:
        args.tag = 'latest'

    # Pull the image if requested, unless the image is specified as a hash
    # that we already have.
    if args.pull and not find_image_hashes(args.image):
        pull_image(args.image, args.tag)

    try:
        image_hash = find_one_image_hash(args.image, args.tag)
    except DockerError as error:
        logger.error(error.message)
        sys.exit(1)

    if not docker_image_compatible(api, image_hash):
        if args.force_image_format:
            logger.warning("forcing incompatible image")
        else:
            logger.error("refusing to store " \
                "incompatible format (use --force-image-format to override)")
            sys.exit(1)

    image_repo_tag = '{}:{}'.format(args.image, args.tag) if not image_hash.startswith(args.image.lower()) else None

    if args.name is None:
        if image_repo_tag:
            collection_name = 'Docker image {} {}'.format(image_repo_tag, image_hash[0:12])
        else:
            collection_name = 'Docker image {}'.format(image_hash[0:12])
    else:
        collection_name = args.name

    if not args.force:
        # Check if this image is already in Arvados.

        # Project where everything should be owned
        if args.project_uuid:
            parent_project_uuid = args.project_uuid
        else:
            parent_project_uuid = api.users().current().execute(
                num_retries=args.retries)['uuid']

        # Find image hash tags
        existing_links = _get_docker_links(
            api, args.retries,
            filters=[['link_class', '=', 'docker_image_hash'],
                     ['name', '=', image_hash]])
        if existing_links:
            # get readable collections
            collections = api.collections().list(
                filters=[['uuid', 'in', [link['head_uuid'] for link in existing_links]]],
                select=["uuid", "owner_uuid", "name", "manifest_text"]
                ).execute(num_retries=args.retries)['items']

            if collections:
                # check for repo+tag links on these collections
                if image_repo_tag:
                    existing_repo_tag = _get_docker_links(
                        api, args.retries,
                        filters=[['link_class', '=', 'docker_image_repo+tag'],
                                 ['name', '=', image_repo_tag],
                                 ['head_uuid', 'in', [c["uuid"] for c in collections]]])
                else:
                    existing_repo_tag = []

                try:
                    coll_uuid = next(items_owned_by(parent_project_uuid, collections))['uuid']
                except StopIteration:
                    # create new collection owned by the project
                    coll_uuid = api.collections().create(
                        body={"manifest_text": collections[0]['manifest_text'],
                              "name": collection_name,
                              "owner_uuid": parent_project_uuid},
                        ensure_unique_name=True
                        ).execute(num_retries=args.retries)['uuid']

                link_base = {'owner_uuid': parent_project_uuid,
                             'head_uuid':  coll_uuid,
                             'properties': existing_links[0]['properties']}

                if not any(items_owned_by(parent_project_uuid, existing_links)):
                    # create image link owned by the project
                    make_link(api, args.retries,
                              'docker_image_hash', image_hash, **link_base)

                if image_repo_tag and not any(items_owned_by(parent_project_uuid, existing_repo_tag)):
                    # create repo+tag link owned by the project
                    make_link(api, args.retries, 'docker_image_repo+tag',
                              image_repo_tag, **link_base)

                stdout.write(coll_uuid + "\n")

                sys.exit(0)

    # Open a file for the saved image, and write it if needed.
    outfile_name = '{}.tar'.format(image_hash)
    image_file, need_save = prep_image_file(outfile_name)
    if need_save:
        save_image(image_hash, image_file)

    # Call arv-put with switches we inherited from it
    # (a.k.a., switches that aren't our own).
    put_args = keepdocker_parser.parse_known_args(arguments)[1]

    if args.name is None:
        put_args += ['--name', collection_name]

    coll_uuid = arv_put.main(
        put_args + ['--filename', outfile_name, image_file.name], stdout=stdout).strip()

    # Read the image metadata and make Arvados links from it.
    image_file.seek(0)
    image_tar = tarfile.open(fileobj=image_file)
    image_hash_type, _, raw_image_hash = image_hash.rpartition(':')
    if image_hash_type:
        json_filename = raw_image_hash + '.json'
    else:
        json_filename = raw_image_hash + '/json'
    json_file = image_tar.extractfile(image_tar.getmember(json_filename))
    image_metadata = json.load(json_file)
    json_file.close()
    image_tar.close()
    link_base = {'head_uuid': coll_uuid, 'properties': {}}
    if 'created' in image_metadata:
        link_base['properties']['image_timestamp'] = image_metadata['created']
    if args.project_uuid is not None:
        link_base['owner_uuid'] = args.project_uuid

    make_link(api, args.retries, 'docker_image_hash', image_hash, **link_base)
    if image_repo_tag:
        make_link(api, args.retries,
                  'docker_image_repo+tag', image_repo_tag, **link_base)

    # Clean up.
    image_file.close()
    for filename in [stat_cache_name(image_file), image_file.name]:
        try:
            os.unlink(filename)
        except OSError as error:
            if error.errno != errno.ENOENT:
                raise

if __name__ == '__main__':
    main()
