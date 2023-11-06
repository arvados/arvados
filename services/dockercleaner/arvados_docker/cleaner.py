#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

"""arvados_docker.cleaner - Remove unused Docker images from compute nodes

Usage:
  python3 -m arvados_docker.cleaner --quota 50G
"""

import argparse
import collections
import copy
import functools
import json
import logging
import sys
import time

import docker
import json

DEFAULT_CONFIG_FILE = '/etc/arvados/docker-cleaner/docker-cleaner.json'

SUFFIX_SIZES = {suffix: 1024 ** exp for exp, suffix in enumerate('kmgt', 1)}

logger = logging.getLogger('arvados_docker.cleaner')


def return_when_docker_not_found(result=None):
    # If the decorated function raises a 404 error from Docker, return
    # `result` instead.
    def docker_not_found_decorator(orig_func):
        @functools.wraps(orig_func)
        def docker_not_found_wrapper(*args, **kwargs):
            try:
                return orig_func(*args, **kwargs)
            except docker.errors.APIError as error:
                if error.response.status_code != 404:
                    raise
                return result
        return docker_not_found_wrapper
    return docker_not_found_decorator


class DockerImage:

    def __init__(self, image_hash):
        self.docker_id = image_hash['Id']
        self.size = image_hash['VirtualSize']
        self.last_used = -1

    def used_at(self, use_time):
        self.last_used = max(self.last_used, use_time)


class DockerImages:

    def __init__(self, target_size):
        self.target_size = target_size
        self.images = {}
        self.container_image_map = {}

    @classmethod
    def from_daemon(cls, target_size, docker_client):
        images = cls(target_size)
        for image in docker_client.images():
            images.add_image(image)
        return images

    def add_image(self, image_hash):
        image = DockerImage(image_hash)
        self.images[image.docker_id] = image
        logger.debug("Registered image %s", image.docker_id)

    def del_image(self, image_id):
        if image_id in self.images:
            del self.images[image_id]
            self.container_image_map = {
                cid: cid_image
                for cid, cid_image in self.container_image_map.items()
                if cid_image != image_id}
            logger.debug("Unregistered image %s", image_id)

    def has_image(self, image_id):
        return image_id in self.images

    def add_user(self, container_hash, use_time):
        image_id = container_hash['Image']
        if image_id in self.images:
            self.container_image_map[container_hash['Id']] = image_id
            self.images[image_id].used_at(use_time)
            logger.debug("Registered container %s using image %s",
                         container_hash['Id'], image_id)

    def end_user(self, cid):
        self.container_image_map.pop(cid, None)
        logger.debug("Unregistered container %s", cid)

    def should_delete(self):
        if not self.images:
            return
        # Build a list of images, ordered by use time.
        lru_images = list(self.images.values())
        lru_images.sort(key=lambda image: image.last_used)
        # Make sure we don't delete any images in use, or if there are
        # none, the most recently used image.
        if self.container_image_map:
            keep_ids = set(self.container_image_map.values())
        else:
            keep_ids = {lru_images[-1].docker_id}
        space_left = (self.target_size - sum(self.images[image_id].size
                                             for image_id in keep_ids))
        # Go through the list most recently used first, and note which
        # images can be saved with the space allotted.
        for image in reversed(lru_images):
            if (image.docker_id not in keep_ids) and (image.size <= space_left):
                keep_ids.add(image.docker_id)
                space_left -= image.size
        # Yield the Docker IDs of any image we don't want to save, least
        # recently used first.
        for image in lru_images:
            if image.docker_id not in keep_ids:
                yield image.docker_id


class DockerEventHandlers:
    # This class maps Docker event types to the names of methods that should
    # receive those events.

    def __init__(self):
        self.handler_names = collections.defaultdict(list)

    def on(self, *status_names):
        def register_handler(handler_method):
            for status in status_names:
                self.handler_names[status].append(handler_method.__name__)
            return handler_method
        return register_handler

    def for_event(self, status):
        return iter(self.handler_names[status])

    def copy(self):
        result = self.__class__()
        result.handler_names = copy.deepcopy(self.handler_names)
        return result


class DockerEventListener:
    # To use this class, define event_handlers as an instance of
    # DockerEventHandlers.  Call run() to iterate over events and call the
    # handler methods as they come in.
    ENCODING = 'utf-8'

    def __init__(self, events):
        self.events = events

    def run(self):
        for event in self.events:
            event = json.loads(event.decode(self.ENCODING))
            if event.get('Type', 'container') != 'container':
                continue
            for method_name in self.event_handlers.for_event(event.get('status')):
                getattr(self, method_name)(event)


class DockerImageUseRecorder(DockerEventListener):
    event_handlers = DockerEventHandlers()

    def __init__(self, images, docker_client, events):
        self.images = images
        self.docker_client = docker_client
        super().__init__(events)

    @event_handlers.on('create')
    @return_when_docker_not_found()
    def load_container(self, event):
        container_hash = self.docker_client.inspect_container(event['id'])
        self.new_container(event, container_hash)

    def new_container(self, event, container_hash):
        self.images.add_user(container_hash, event['time'])

    @event_handlers.on('destroy')
    def container_stopped(self, event):
        self.images.end_user(event['id'])


class DockerImageCleaner(DockerImageUseRecorder):
    event_handlers = DockerImageUseRecorder.event_handlers.copy()

    def __init__(self, images, docker_client, events, remove_containers_onexit=False):
        super().__init__(images, docker_client, events)
        self.logged_unknown = set()
        self.remove_containers_onexit = remove_containers_onexit

    def new_container(self, event, container_hash):
        container_image_id = container_hash['Image']
        if not self.images.has_image(container_image_id):
            image_hash = self.docker_client.inspect_image(container_image_id)
            self.images.add_image(image_hash)
        return super().new_container(event, container_hash)

    def _remove_container(self, cid):
        try:
            self.docker_client.remove_container(cid, v=True)
        except docker.errors.APIError as error:
            logger.warning("Failed to remove container %s: %s", cid, error)
        else:
            logger.info("Removed container %s", cid)

    @event_handlers.on('die')
    def clean_container(self, event=None):
        if self.remove_containers_onexit:
            self._remove_container(event['id'])

    def check_stopped_containers(self, remove=False):
        logger.info("Checking for stopped containers")
        for c in self.docker_client.containers(filters={'status': 'exited'}):
            logger.info("Container %s %s", c['Id'], c['Status'])
            if c['Status'][:6] != 'Exited':
                logger.error("Unexpected status %s for container %s",
                             c['Status'], c['Id'])
            elif remove:
                self._remove_container(c['Id'])

    @event_handlers.on('destroy')
    def clean_images(self, event=None):
        for image_id in self.images.should_delete():
            try:
                self.docker_client.remove_image(image_id)
            except docker.errors.APIError as error:
                logger.warning(
                    "Failed to remove image %s: %s", image_id, error)
            else:
                logger.info("Removed image %s", image_id)
                self.images.del_image(image_id)

    @event_handlers.on('destroy')
    def log_unknown_images(self, event):
        unknown_ids = {image['Id'] for image in self.docker_client.images()
                       if not self.images.has_image(image['Id'])}
        for image_id in (unknown_ids - self.logged_unknown):
            logger.info(
                "Image %s is loaded but unused, so it won't be cleaned",
                image_id)
        self.logged_unknown = unknown_ids


def human_size(size_str):
    size_str = size_str.lower().rstrip('b')
    multiplier = SUFFIX_SIZES.get(size_str[-1])
    if multiplier is None:
        multiplier = 1
    else:
        size_str = size_str[:-1]
    return int(size_str) * multiplier


def load_config(arguments):
    args = parse_arguments(arguments)

    config = default_config()
    try:
        with open(args.config, 'r') as f:
            c = json.load(f)
            config.update(c)
    except (FileNotFoundError, IOError, ValueError) as error:
        if (isinstance(error, FileNotFoundError) and
            args.config == DEFAULT_CONFIG_FILE):
            logger.warning("DEPRECATED: default config file %s not found; "
                           "relying on command line configuration",
                           repr(DEFAULT_CONFIG_FILE))
        else:
            sys.exit('error reading config file {}: {}'.format(
                args.config, error))

    configargs = vars(args).copy()
    configargs.pop('config')
    config.update({k: v for k, v in configargs.items() if v})

    if isinstance(config['Quota'], str):
        config['Quota'] = human_size(config['Quota'])

    return config


def default_config():
    return {
        'Quota': '1G',
        'RemoveStoppedContainers': 'always',
        'Verbose': 0,
    }


def parse_arguments(arguments):
    class Formatter(argparse.ArgumentDefaultsHelpFormatter,
                    argparse.RawDescriptionHelpFormatter):
        pass
    parser = argparse.ArgumentParser(
        prog="arvados_docker.cleaner",
        description="clean old Docker images from Arvados compute nodes",
        epilog="Example config file:\n\n{}".format(
            json.dumps(default_config(), indent=4)),
        formatter_class=Formatter,
    )
    parser.add_argument(
        '--config', action='store', type=str, default=DEFAULT_CONFIG_FILE,
        help="configuration file")

    deprecated = " (DEPRECATED -- use config file instead)"
    parser.add_argument(
        '--quota', action='store', type=human_size, dest='Quota',
        help="space allowance for Docker images, suffixed with K/M/G/T" + deprecated)
    parser.add_argument(
        '--remove-stopped-containers', type=str, default='always', dest='RemoveStoppedContainers',
        choices=['never', 'onexit', 'always'],
        help="""when to remove stopped containers (default: always, i.e., remove
        stopped containers found at startup, and remove containers as
        soon as they exit)""" + deprecated)
    parser.add_argument(
        '--verbose', '-v', action='count', default=0, dest='Verbose',
        help="log more information" + deprecated)

    return parser.parse_args(arguments)


def setup_logging():
    log_handler = logging.StreamHandler()
    log_handler.setFormatter(logging.Formatter(
        '%(asctime)s %(name)s[%(process)d] %(levelname)s: %(message)s',
        '%Y-%m-%d %H:%M:%S'))
    logger.addHandler(log_handler)


def configure_logging(config):
    logger.setLevel(logging.ERROR - (10 * config['Verbose']))


def run(config, docker_client):
    start_time = int(time.time())
    logger.debug("Loading Docker activity through present")
    images = DockerImages.from_daemon(config['Quota'], docker_client)
    use_recorder = DockerImageUseRecorder(
        images, docker_client, docker_client.events(since=1, until=start_time))
    use_recorder.run()
    cleaner = DockerImageCleaner(
        images, docker_client, docker_client.events(since=start_time),
        remove_containers_onexit=config['RemoveStoppedContainers'] != 'never')
    cleaner.check_stopped_containers(
        remove=config['RemoveStoppedContainers'] == 'always')
    logger.info("Checking image quota at startup")
    cleaner.clean_images()
    logger.info("Listening for docker events")
    cleaner.run()


def main(arguments=sys.argv[1:]):
    setup_logging()
    config = load_config(arguments)
    configure_logging(config)
    try:
        run(config, docker.APIClient(version='1.35'))
    except KeyboardInterrupt:
        sys.exit(1)

if __name__ == '__main__':
    main()
