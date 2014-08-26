#!/usr/bin/env python

import argparse
import errno
import json
import os
import subprocess
import sys
import tarfile
import tempfile

from collections import namedtuple
from stat import *

import arvados
import arvados.commands._util as arv_cmd
import arvados.commands.put as arv_put

STAT_CACHE_ERRORS = (IOError, OSError, ValueError)

DockerImage = namedtuple('DockerImage',
                         ['repo', 'tag', 'hash', 'created', 'vsize'])

opt_parser = argparse.ArgumentParser(add_help=False)
opt_parser.add_argument(
    '-f', '--force', action='store_true', default=False,
    help="Re-upload the image even if it already exists on the server")

_group = opt_parser.add_mutually_exclusive_group()
_group.add_argument(
    '--pull', action='store_true', default=True,
    help="Pull the latest image from Docker repositories first (default)")
_group.add_argument(
    '--no-pull', action='store_false', dest='pull',
    help="Don't pull images from Docker repositories")

opt_parser.add_argument(
    'image',
    help="Docker image to upload, as a repository name or hash")
opt_parser.add_argument(
    'tag', nargs='?', default='latest',
    help="Tag of the Docker image to upload (default 'latest')")

arg_parser = argparse.ArgumentParser(
        description="Upload a Docker image to Arvados",
        parents=[opt_parser, arv_put.run_opts])

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
    check_docker(popen_docker(['pull', '-t', image_tag, image_name]), "pull")

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

def make_link(link_class, link_name, **link_attrs):
    link_attrs.update({'link_class': link_class, 'name': link_name})
    return arvados.api('v1').links().create(body=link_attrs).execute()

def main(arguments=None):
    args = arg_parser.parse_args(arguments)

    # Pull the image if requested, unless the image is specified as a hash
    # that we already have.
    if args.pull and not find_image_hashes(args.image):
        pull_image(args.image, args.tag)

    try:
        image_hash = find_one_image_hash(args.image, args.tag)
    except DockerError as error:
        print >>sys.stderr, "arv-keepdocker:", error.message
        sys.exit(1)
    if not args.force:
        # Abort if this image is already in Arvados.
        existing_links = arvados.api('v1').links().list(
            filters=[['link_class', '=', 'docker_image_hash'],
                     ['name', '=', image_hash]]).execute()['items']
        if existing_links:
            message = [
                "arv-keepdocker: Image {} already stored in collection(s):".
                format(image_hash)]
            message.extend(link['head_uuid'] for link in existing_links)
            print >>sys.stderr, "\n".join(message)
            sys.exit(0)

    # Open a file for the saved image, and write it if needed.
    outfile_name = '{}.tar'.format(image_hash)
    image_file, need_save = prep_image_file(outfile_name)
    if need_save:
        save_image(image_hash, image_file)

    # Call arv-put with switches we inherited from it
    # (a.k.a., switches that aren't our own).
    put_args = opt_parser.parse_known_args(arguments)[1]
    coll_uuid = arv_put.main(
        put_args + ['--filename', outfile_name, image_file.name]).strip()

    # Read the image metadata and make Arvados links from it.
    image_file.seek(0)
    image_tar = tarfile.open(fileobj=image_file)
    json_file = image_tar.extractfile(image_tar.getmember(image_hash + '/json'))
    image_metadata = json.load(json_file)
    json_file.close()
    image_tar.close()
    link_base = {'head_uuid': coll_uuid, 'properties': {}}
    if 'created' in image_metadata:
        link_base['properties']['image_timestamp'] = image_metadata['created']

    make_link('docker_image_hash', image_hash, **link_base)
    if not image_hash.startswith(args.image.lower()):
        make_link('docker_image_repo+tag', '{}:{}'.format(args.image, args.tag),
                  **link_base)

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
