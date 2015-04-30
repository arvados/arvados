#!/usr/bin/env python3
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
            for method_name in self.event_handlers.for_event(event['status']):
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

    def __init__(self, images, docker_client, events):
        super().__init__(images, docker_client, events)
        self.logged_unknown = set()

    def new_container(self, event, container_hash):
        container_image_id = container_hash['Image']
        if not self.images.has_image(container_image_id):
            image_hash = self.docker_client.inspect_image(container_image_id)
            self.images.add_image(image_hash)
        return super().new_container(event, container_hash)

    @event_handlers.on('destroy')
    def clean_images(self, event=None):
        for image_id in self.images.should_delete():
            try:
                self.docker_client.remove_image(image_id)
            except docker.errors.APIError as error:
                logger.warning("Failed to remove image %s: %s", image_id, error)
            else:
                logger.info("Removed image %s", image_id)
                self.images.del_image(image_id)

    @event_handlers.on('destroy')
    def log_unknown_images(self, event):
        unknown_ids = {image['Id'] for image in self.docker_client.images()
                       if not self.images.has_image(image['Id'])}
        for image_id in (unknown_ids - self.logged_unknown):
            logger.info("Image %s is loaded but unused, so it won't be cleaned",
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

def parse_arguments(arguments):
    parser = argparse.ArgumentParser(
        prog="arvados_docker.cleaner",
        description="clean old Docker images from Arvados compute nodes")
    parser.add_argument(
        '--quota', action='store', type=human_size, required=True,
        help="space allowance for Docker images, suffixed with K/M/G/T")
    parser.add_argument(
        '--verbose', '-v', action='count', default=0,
        help="log more information")
    return parser.parse_args(arguments)

def setup_logging(args):
    log_handler = logging.StreamHandler()
    log_handler.setFormatter(logging.Formatter(
            '%(asctime)s %(name)s[%(process)d] %(levelname)s: %(message)s',
            '%Y-%m-%d %H:%M:%S'))
    logger.addHandler(log_handler)
    logger.setLevel(logging.ERROR - (10 * args.verbose))

def run(args, docker_client):
    start_time = int(time.time())
    logger.debug("Loading Docker activity through present")
    images = DockerImages.from_daemon(args.quota, docker_client)
    use_recorder = DockerImageUseRecorder(
        images, docker_client, docker_client.events(since=1, until=start_time))
    use_recorder.run()
    cleaner = DockerImageCleaner(
        images, docker_client, docker_client.events(since=start_time))
    logger.info("Starting cleanup loop")
    cleaner.clean_images()
    cleaner.run()

def main(arguments):
    args = parse_arguments(arguments)
    setup_logging(args)
    run(args, docker.Client())

if __name__ == '__main__':
    main(sys.argv[1:])
