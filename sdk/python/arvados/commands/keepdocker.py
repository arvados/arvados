#!/usr/bin/env python

import argparse
import datetime
import errno
import json
import os
import re
import subprocess
import sys
import tarfile
import tempfile
import _strptime

from collections import namedtuple
from stat import *

import arvados
from arvados.util import list_all
import arvados.commands._util as arv_cmd
import arvados.commands.put as arv_put
import arvados.errors

STAT_CACHE_ERRORS = (IOError, OSError, ValueError)

DockerImage = namedtuple('DockerImage',
                         ['repo', 'tag', 'hash', 'created', 'vsize'])

keepdocker_parser = argparse.ArgumentParser(add_help=False)
keepdocker_parser.add_argument(
    '-f', '--force', action='store_true', default=False,
    help="Re-upload the image even if it already exists on the server")

keepdocker_parser.add_argument(
    '--no-trunc', action='store_true', default=False,
    help="Don't truncate Docker image hashes in output.")

_group = keepdocker_parser.add_mutually_exclusive_group()
_group.add_argument(
    '--pull', action='store_true', default=False,
    help="Try to pull the latest image from Docker registry")
_group.add_argument(
    '--no-pull', action='store_false', dest='pull',
    help="Use locally installed image only, don't pull image from Docker registry (default)")

_group = keepdocker_parser.add_mutually_exclusive_group()
_group.add_argument(
    '--download', action='store_true', default=False,
    help="Fetch Docker image from Arvados and load locally.")
_group.add_argument(
    '--upload', action='store_true', default=False,
    help="Upload local Docker image to Arvados (default)")

keepdocker_parser.add_argument(
    'image', nargs='?',
    help="Docker image as a repository name or hash")
keepdocker_parser.add_argument(
    'tag', nargs='?', default='latest',
    help="Tag of the Docker image to upload (default 'latest')")

# Combine keepdocker options listed above with run_opts options of arv-put.
# The options inherited from arv-put include --name, --project-uuid,
# --progress/--no-progress/--batch-progress and --resume/--no-resume.
arg_parser = argparse.ArgumentParser(
        description="Upload, download or list Docker images in Arvados",
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

def ptimestamp(t):
    s = t.split(".")
    if len(s) == 2:
        t = s[0] + s[1][-1:]
    return datetime.datetime.strptime(t, "%Y-%m-%dT%H:%M:%SZ")

def list_images_in_arv(api_client, num_retries, image_name=None, image_tag=None, image_hash=None, image_collection=None):
    """List all Docker images known to the api_client with image_name and
    image_tag.  If no image_name is given, defaults to listing all
    Docker images.

    Returns a list of tuples representing matching Docker images,
    sorted in preference order (i.e. the first collection in the list
    is the one that the API server would use). Each tuple is a
    (collection_uuid, collection_info) pair, where collection_info is
    a dict with fields "dockerhash", "repo", "tag", and "timestamp".

    """
    docker_image_filters = [['link_class', 'in', ['docker_image_hash', 'docker_image_repo+tag']]]
    if image_name:
        image_link_name = "{}:{}".format(image_name, image_tag or 'latest')
        docker_image_filters.append(['name', '=', image_link_name])
    if image_hash:
        docker_image_filters.append(['name', '=', image_hash])
    if image_collection:
        docker_image_filters.append(['head_uuid', '=', image_collection])

    existing_links = list_all(api_client.links().list, num_retries, filters=docker_image_filters)
    images = {}
    for link in existing_links:
        collection_uuid = link["head_uuid"]
        if collection_uuid not in images:
            images[collection_uuid]= {"dockerhash": "<none>",
                      "repo":"<none>",
                      "tag":"<none>",
                      "timestamp": ptimestamp("1970-01-01T00:00:01Z")}

        if link["link_class"] == "docker_image_hash":
            images[collection_uuid]["dockerhash"] = link["name"]

        if link["link_class"] == "docker_image_repo+tag":
            r = link["name"].split(":")
            images[collection_uuid]["repo"] = r[0]
            if len(r) > 1:
                images[collection_uuid]["tag"] = r[1]

        if "image_timestamp" in link["properties"]:
            images[collection_uuid]["timestamp"] = ptimestamp(link["properties"]["image_timestamp"])
        else:
            images[collection_uuid]["timestamp"] = ptimestamp(link["created_at"])

    return sorted(images.items(), lambda a, b: cmp(b[1]["timestamp"], a[1]["timestamp"]))

def image_hash_in_collection(cr):
    if len(cr) != 1:
        raise arvados.errors.ArgumentError("docker_image_locator must only contain a single file")

    docker_image = re.match("([0-9a-f]{64})\.tar", cr.keys()[0])
    if docker_image:
        return docker_image.group(1)
    else:
        return None

def load_image_from_collection(api_client, docker_image_locator):
    cr = arvados.CollectionReader(docker_image_locator, api_client=api_client)
    docker_image = image_hash_in_collection(cr)
    if docker_image:
        for d in docker_images():
            if d.hash == docker_image:
                print "Docker image '%s' is already loaded" % docker_image
                return docker_image

        with cr.open(docker_image+".tar") as img:
            docker_load = subprocess.Popen(["docker", "load"], stdin=subprocess.PIPE)
            data = img.read(64000)
            n = len(data)
            while data:
                docker_load.stdin.write(data)
                data = img.read(1024*1024)
                n += len(data)
        docker_load.stdin.close()
        docker_load.wait()
        if docker_load.returncode != 0:
            raise arvados.errors.CommandFailedError("Failed to load image")

        return docker_image
    else:
        raise arvados.errors.ArgumentError("Failed to find Docker image in collection %s" % docker_image_locator)


def main(arguments=None):
    args = arg_parser.parse_args(arguments)
    api = arvados.api('v1')

    if args.image is None or args.image == 'images':
        if args.no_trunc:
            fmt = "{:30}  {:10}  {:64}  {:29}  {:20}"
        else:
            fmt = "{:30}  {:10}  {:12}  {:29}  {:20}"
        print fmt.format("REPOSITORY", "TAG", "IMAGE ID", "COLLECTION", "CREATED")
        for i, j in list_images_in_arv(api, args.retries):
            print(fmt.format(j["repo"], j["tag"],
                             j["dockerhash"] if args.no_trunc else j["dockerhash"][0:12],
                             i, j["timestamp"].strftime("%c")))
        sys.exit(0)

    if args.download:
        # search by name and tag
        imgs_in_arv = list_images_in_arv(api, args.retries, image_name=args.image)
        do_tag = True

        if not imgs_in_arv:
            # searh by image hash
            imgs_in_arv = list_images_in_arv(api, args.retries, image_hash=args.image)
            do_tag = False

        if not imgs_in_arv and arvados.util.collection_uuid_pattern.match(args.image):
            # search by collection uuid
            imgs_in_arv = list_images_in_arv(api, args.retries, image_collection=args.image)
            do_tag = True

        if not imgs_in_arv and arvados.util.keep_locator_pattern.match(args.image):
            # search by manifest portable data hash
            imgs_in_arv = [[args.image]]
            do_tag = False

        if imgs_in_arv:
            imghash = load_image_from_collection(api, imgs_in_arv[0][0])
            if do_tag:
                popen_docker(["tag", imghash, args.image], stdin=None, stdout=None).wait()
            sys.exit(0)
        else:
            print >>sys.stderr, "arv-keepdocker: Docker image '%s' not found in Arvados" % args.image
            sys.exit(1)

    # Pull the image if requested, unless the image is specified as a hash
    # that we already have.
    if args.pull and not find_image_hashes(args.image):
        pull_image(args.image, args.tag)

    try:
        image_hash = find_one_image_hash(args.image, args.tag)
    except DockerError as error:
        print >>sys.stderr, "arv-keepdocker:", error.message
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
        existing_links = list_all(api.links().list, num_retries=args.retries,
                                  filters=[['link_class', '=', 'docker_image_hash'],
                                           ['name', '=', image_hash]])
        if existing_links:
            # get readable collections
            collections = list_all(api.collections().list, num_retries=args.retries,
                                   filters=[['uuid', 'in', [link['head_uuid'] for link in existing_links]]],
                                   select=["uuid", "owner_uuid", "name", "manifest_text"])

            if collections:
                # check for repo+tag links on these collections
                existing_repo_tag = list_all(api.links().list, num_retries=args.retries,
                    filters=[['link_class', '=', 'docker_image_repo+tag'],
                             ['name', '=', image_repo_tag],
                             ['head_uuid', 'in', collections]]) if image_repo_tag else []

                # Filter on elements owned by the parent project
                owned_col = [c for c in collections if c['owner_uuid'] == parent_project_uuid]
                owned_img = [c for c in existing_links if c['owner_uuid'] == parent_project_uuid]
                owned_rep = [c for c in existing_repo_tag if c['owner_uuid'] == parent_project_uuid]

                if owned_col:
                    # already have a collection owned by this project
                    coll_uuid = owned_col[0]['uuid']
                else:
                    # create new collection owned by the project
                    coll_uuid = api.collections().create(
                        body={"manifest_text": collections[0]['manifest_text'],
                              "name": collection_name,
                              "owner_uuid": parent_project_uuid},
                        ensure_unique_name=True
                        ).execute(num_retries=args.retries)['uuid']

                link_base = {'owner_uuid': parent_project_uuid,
                             'head_uuid':  coll_uuid }

                if not owned_img:
                    # create image link owned by the project
                    make_link(api, args.retries,
                              'docker_image_hash', image_hash, **link_base)

                if not owned_rep and image_repo_tag:
                    # create repo+tag link owned by the project
                    make_link(api, args.retries, 'docker_image_repo+tag',
                              image_repo_tag, **link_base)

                print(coll_uuid)

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
