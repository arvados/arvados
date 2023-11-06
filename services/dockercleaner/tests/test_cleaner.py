#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import collections
import itertools
import json
import os
import random
import tempfile
import time
import unittest

import docker
from unittest import mock

from arvados_docker import cleaner

MAX_DOCKER_ID = (16 ** 64) - 1


def MockDockerId():
    return '{:064x}'.format(random.randint(0, MAX_DOCKER_ID))


def MockContainer(image_hash):
    return {'Id': MockDockerId(),
            'Image': image_hash['Id']}


def MockImage(*, size=0, vsize=None, tags=[]):
    if vsize is None:
        vsize = random.randint(100, 2000000)
    return {'Id': MockDockerId(),
            'ParentId': MockDockerId(),
            'RepoTags': list(tags),
            'Size': size,
            'VirtualSize': vsize}


class MockEvent(dict):
    ENCODING = 'utf-8'
    event_seq = itertools.count(1)

    def __init__(self, status, docker_id=None, **event_data):
        if docker_id is None:
            docker_id = MockDockerId()
        super().__init__(self, **event_data)
        self['status'] = status
        self['id'] = docker_id
        self.setdefault('time', next(self.event_seq))

    def encoded(self):
        return json.dumps(self).encode(self.ENCODING)


class MockException(docker.errors.APIError):

    def __init__(self, status_code):
        response = mock.Mock(name='response')
        response.status_code = status_code
        super().__init__("mock exception", response)


class DockerImageTestCase(unittest.TestCase):

    def test_used_at_sets_last_used(self):
        image = cleaner.DockerImage(MockImage())
        image.used_at(5)
        self.assertEqual(5, image.last_used)

    def test_used_at_moves_forward(self):
        image = cleaner.DockerImage(MockImage())
        image.used_at(6)
        image.used_at(8)
        self.assertEqual(8, image.last_used)

    def test_used_at_does_not_go_backward(self):
        image = cleaner.DockerImage(MockImage())
        image.used_at(4)
        image.used_at(2)
        self.assertEqual(4, image.last_used)


class DockerImagesTestCase(unittest.TestCase):

    def setUp(self):
        self.mock_images = []

    def setup_mock_images(self, *vsizes):
        self.mock_images.extend(MockImage(vsize=vsize) for vsize in vsizes)

    def setup_images(self, *vsizes, target_size=1000000):
        self.setup_mock_images(*vsizes)
        images = cleaner.DockerImages(target_size)
        for image in self.mock_images:
            images.add_image(image)
        return images

    def test_has_image(self):
        images = self.setup_images(None)
        self.assertTrue(images.has_image(self.mock_images[0]['Id']))
        self.assertFalse(images.has_image(MockDockerId()))

    def test_del_image(self):
        images = self.setup_images(None)
        images.del_image(self.mock_images[0]['Id'])
        self.assertFalse(images.has_image(self.mock_images[0]['Id']))

    def test_del_nonexistent_image(self):
        images = self.setup_images(None)
        images.del_image(MockDockerId())
        self.assertTrue(images.has_image(self.mock_images[0]['Id']))

    def test_one_image_always_kept(self):
        # When crunch-job starts a job, it makes sure each compute node
        # has the Docker image loaded, then it runs all the tasks with
        # the assumption the image is on each node.  As long as that's
        # true, the cleaner should avoid removing every installed image:
        # crunch-job might be counting on the most recent one to be
        # available, even if it's not currently in use.
        images = self.setup_images(None, None, target_size=1)
        for use_time, image in enumerate(self.mock_images, 1):
            user = MockContainer(image)
            images.add_user(user, use_time)
            images.end_user(user['Id'])
        self.assertEqual([self.mock_images[0]['Id']],
                         list(images.should_delete()))

    def test_images_under_target_not_deletable(self):
        # The images are used in this order.  target_size is set so it
        # could hold the largest image, but not after the most recently
        # used image is kept; then we have to fall back to the previous one.
        images = self.setup_images(20, 30, 40, 10, target_size=45)
        for use_time, image in enumerate(self.mock_images, 1):
            user = MockContainer(image)
            images.add_user(user, use_time)
            images.end_user(user['Id'])
        self.assertEqual([self.mock_images[ii]['Id'] for ii in [0, 2]],
                         list(images.should_delete()))

    def test_images_in_use_not_deletable(self):
        images = self.setup_images(None, None, target_size=1)
        users = [MockContainer(image) for image in self.mock_images]
        images.add_user(users[0], 1)
        images.add_user(users[1], 2)
        images.end_user(users[1]['Id'])
        self.assertEqual([self.mock_images[1]['Id']],
                         list(images.should_delete()))

    def test_image_deletable_after_unused(self):
        images = self.setup_images(None, None, target_size=1)
        users = [MockContainer(image) for image in self.mock_images]
        images.add_user(users[0], 1)
        images.add_user(users[1], 2)
        images.end_user(users[0]['Id'])
        self.assertEqual([self.mock_images[0]['Id']],
                         list(images.should_delete()))

    def test_image_not_deletable_if_user_restarts(self):
        images = self.setup_images(None, target_size=1)
        user = MockContainer(self.mock_images[-1])
        images.add_user(user, 1)
        images.end_user(user['Id'])
        images.add_user(user, 2)
        self.assertEqual([], list(images.should_delete()))

    def test_image_not_deletable_if_any_user_remains(self):
        images = self.setup_images(None, target_size=1)
        users = [MockContainer(self.mock_images[0]) for ii in range(2)]
        images.add_user(users[0], 1)
        images.add_user(users[1], 2)
        images.end_user(users[0]['Id'])
        self.assertEqual([], list(images.should_delete()))

    def test_image_deletable_after_all_users_end(self):
        images = self.setup_images(None, None, target_size=1)
        users = [MockContainer(self.mock_images[ii]) for ii in [0, 1, 1]]
        images.add_user(users[0], 1)
        images.add_user(users[1], 2)
        images.add_user(users[2], 3)
        images.end_user(users[1]['Id'])
        images.end_user(users[2]['Id'])
        self.assertEqual([self.mock_images[-1]['Id']],
                         list(images.should_delete()))

    def test_images_suggested_for_deletion_by_lru(self):
        images = self.setup_images(10, 10, 10, target_size=1)
        users = [MockContainer(image) for image in self.mock_images]
        images.add_user(users[0], 3)
        images.add_user(users[1], 1)
        images.add_user(users[2], 2)
        for user in users:
            images.end_user(user['Id'])
        self.assertEqual([self.mock_images[ii]['Id'] for ii in [1, 2]],
                         list(images.should_delete()))

    def test_adding_user_without_image_does_not_implicitly_add_image(self):
        images = self.setup_images(10)
        images.add_user(MockContainer(MockImage()), 1)
        self.assertEqual([], list(images.should_delete()))

    def test_nonexistent_user_removed(self):
        images = self.setup_images()
        images.end_user('nonexistent')
        # No exception should be raised.

    def test_del_image_effective_with_users_present(self):
        images = self.setup_images(None, target_size=1)
        user = MockContainer(self.mock_images[0])
        images.add_user(user, 1)
        images.del_image(self.mock_images[0]['Id'])
        images.end_user(user['Id'])
        self.assertEqual([], list(images.should_delete()))

    def setup_from_daemon(self, *vsizes, target_size=1500000):
        self.setup_mock_images(*vsizes)
        docker_client = mock.MagicMock(name='docker_client')
        docker_client.images.return_value = iter(self.mock_images)
        return cleaner.DockerImages.from_daemon(target_size, docker_client)

    def test_images_loaded_from_daemon(self):
        images = self.setup_from_daemon(None, None)
        for image in self.mock_images:
            self.assertTrue(images.has_image(image['Id']))

    def test_target_size_set_from_daemon(self):
        images = self.setup_from_daemon(20, 10, 5, target_size=15)
        user = MockContainer(self.mock_images[-1])
        images.add_user(user, 1)
        self.assertEqual([self.mock_images[0]['Id']],
                         list(images.should_delete()))


class DockerImageUseRecorderTestCase(unittest.TestCase):
    TEST_CLASS = cleaner.DockerImageUseRecorder
    TEST_CLASS_INIT_KWARGS = {}

    def setUp(self):
        self.images = mock.MagicMock(name='images')
        self.docker_client = mock.MagicMock(name='docker_client')
        self.events = []
        self.recorder = self.TEST_CLASS(self.images, self.docker_client,
                                        self.encoded_events, **self.TEST_CLASS_INIT_KWARGS)

    @property
    def encoded_events(self):
        return (event.encoded() for event in self.events)

    def test_unknown_events_ignored(self):
        self.events.append(MockEvent('mock!event'))
        self.recorder.run()
        # No exception should be raised.

    def test_fetches_container_on_create(self):
        self.events.append(MockEvent('create'))
        self.recorder.run()
        self.docker_client.inspect_container.assert_called_with(
            self.events[0]['id'])

    def test_adds_user_on_container_create(self):
        self.events.append(MockEvent('create'))
        self.recorder.run()
        self.images.add_user.assert_called_with(
            self.docker_client.inspect_container(), self.events[0]['time'])

    def test_unknown_image_handling(self):
        # The use recorder should not fetch any images.
        self.events.append(MockEvent('create'))
        self.recorder.run()
        self.assertFalse(self.docker_client.inspect_image.called)

    def test_unfetchable_containers_ignored(self):
        self.events.append(MockEvent('create'))
        self.docker_client.inspect_container.side_effect = MockException(404)
        self.recorder.run()
        self.assertFalse(self.images.add_user.called)

    def test_ends_user_on_container_destroy(self):
        self.events.append(MockEvent('destroy'))
        self.recorder.run()
        self.images.end_user.assert_called_with(self.events[0]['id'])


class DockerImageCleanerTestCase(DockerImageUseRecorderTestCase):
    TEST_CLASS = cleaner.DockerImageCleaner

    def test_unknown_image_handling(self):
        # The image cleaner should fetch and record new images.
        self.images.has_image.return_value = False
        self.events.append(MockEvent('create'))
        self.recorder.run()
        self.docker_client.inspect_image.assert_called_with(
            self.docker_client.inspect_container()['Image'])
        self.images.add_image.assert_called_with(
            self.docker_client.inspect_image())

    def test_unfetchable_images_ignored(self):
        self.images.has_image.return_value = False
        self.docker_client.inspect_image.side_effect = MockException(404)
        self.events.append(MockEvent('create'))
        self.recorder.run()
        self.docker_client.inspect_image.assert_called_with(
            self.docker_client.inspect_container()['Image'])
        self.assertFalse(self.images.add_image.called)

    def test_deletions_after_destroy(self):
        delete_id = MockDockerId()
        self.images.should_delete.return_value = [delete_id]
        self.events.append(MockEvent('destroy'))
        self.recorder.run()
        self.docker_client.remove_image.assert_called_with(delete_id)
        self.images.del_image.assert_called_with(delete_id)

    def test_failed_deletion_handling(self):
        delete_id = MockDockerId()
        self.images.should_delete.return_value = [delete_id]
        self.docker_client.remove_image.side_effect = MockException(500)
        self.events.append(MockEvent('destroy'))
        self.recorder.run()
        self.docker_client.remove_image.assert_called_with(delete_id)
        self.assertFalse(self.images.del_image.called)


class DockerContainerCleanerTestCase(DockerImageUseRecorderTestCase):
    TEST_CLASS = cleaner.DockerImageCleaner
    TEST_CLASS_INIT_KWARGS = {'remove_containers_onexit': True}

    def test_container_deletion_deletes_volumes(self):
        cid = MockDockerId()
        self.events.append(MockEvent('die', docker_id=cid))
        self.recorder.run()
        self.docker_client.remove_container.assert_called_with(cid, v=True)

    @mock.patch('arvados_docker.cleaner.logger')
    def test_failed_container_deletion_handling(self, mockLogger):
        cid = MockDockerId()
        self.docker_client.remove_container.side_effect = MockException(500)
        self.events.append(MockEvent('die', docker_id=cid))
        self.recorder.run()
        self.docker_client.remove_container.assert_called_with(cid, v=True)
        self.assertEqual("Failed to remove container %s: %s",
                         mockLogger.warning.call_args[0][0])
        self.assertEqual(cid,
                         mockLogger.warning.call_args[0][1])


class HumanSizeTestCase(unittest.TestCase):

    def check(self, human_str, count, exp):
        self.assertEqual(count * (1024 ** exp),
                         cleaner.human_size(human_str))

    def test_bytes(self):
        self.check('1', 1, 0)
        self.check('82', 82, 0)

    def test_kibibytes(self):
        self.check('2K', 2, 1)
        self.check('3k', 3, 1)

    def test_mebibytes(self):
        self.check('4M', 4, 2)
        self.check('5m', 5, 2)

    def test_gibibytes(self):
        self.check('6G', 6, 3)
        self.check('7g', 7, 3)

    def test_tebibytes(self):
        self.check('8T', 8, 4)
        self.check('9t', 9, 4)


class RunTestCase(unittest.TestCase):

    def setUp(self):
        self.config = cleaner.default_config()
        self.config['Quota'] = 1000000
        self.docker_client = mock.MagicMock(name='docker_client')

    def test_run(self):
        test_start_time = int(time.time())
        self.docker_client.events.return_value = []
        cleaner.run(self.config, self.docker_client)
        self.assertEqual(2, self.docker_client.events.call_count)
        event_kwargs = [args[1] for args in
                        self.docker_client.events.call_args_list]
        self.assertIn('since', event_kwargs[0])
        self.assertIn('until', event_kwargs[0])
        self.assertLessEqual(test_start_time, event_kwargs[0]['until'])
        self.assertIn('since', event_kwargs[1])
        self.assertEqual(event_kwargs[0]['until'], event_kwargs[1]['since'])


@mock.patch('docker.APIClient', name='docker_client')
@mock.patch('arvados_docker.cleaner.run', name='cleaner_run')
class MainTestCase(unittest.TestCase):

    def test_client_api_version(self, run_mock, docker_client):
        with tempfile.NamedTemporaryFile(mode='wt') as cf:
            cf.write('{"Quota":"1000T"}')
            cf.flush()
            cleaner.main(['--config', cf.name])
        self.assertEqual(1, docker_client.call_count)
        # We are standardized on Docker API version 1.35.
        # See DockerAPIVersion in lib/crunchrun/docker.go.
        self.assertEqual('1.35',
                         docker_client.call_args[1].get('version'))
        self.assertEqual(1, run_mock.call_count)
        self.assertIs(run_mock.call_args[0][1], docker_client())


class ConfigTestCase(unittest.TestCase):

    def test_load_config(self):
        with tempfile.NamedTemporaryFile(mode='wt') as cf:
            cf.write(
                '{"Quota":"1000T", "RemoveStoppedContainers":"always", "Verbose":2}')
            cf.flush()
            config = cleaner.load_config(['--config', cf.name])
        self.assertEqual(1000 << 40, config['Quota'])
        self.assertEqual("always", config['RemoveStoppedContainers'])
        self.assertEqual(2, config['Verbose'])

    def test_args_override_config(self):
        with tempfile.NamedTemporaryFile(mode='wt') as cf:
            cf.write(
                '{"Quota":"1000T", "RemoveStoppedContainers":"always", "Verbose":2}')
            cf.flush()
            config = cleaner.load_config([
                '--config', cf.name,
                '--quota', '1G',
                '--remove-stopped-containers', 'never',
                '--verbose',
            ])
        self.assertEqual(1 << 30, config['Quota'])
        self.assertEqual('never', config['RemoveStoppedContainers'])
        self.assertEqual(1, config['Verbose'])

    def test_args_no_config(self):
        self.assertEqual(False, os.path.exists(cleaner.DEFAULT_CONFIG_FILE))
        config = cleaner.load_config(['--quota', '1G'])
        self.assertEqual(1 << 30, config['Quota'])


class ContainerRemovalTestCase(unittest.TestCase):
    LIFECYCLE = ['create', 'attach', 'start', 'resize', 'die', 'destroy']

    def setUp(self):
        self.config = cleaner.default_config()
        self.docker_client = mock.MagicMock(name='docker_client')
        self.existingCID = MockDockerId()
        self.docker_client.containers.return_value = [{
            'Id': self.existingCID,
            'Status': 'Exited (0) 6 weeks ago',
        }, {
            # If docker_client.containers() returns non-exited
            # containers for some reason, do not remove them.
            'Id': MockDockerId(),
            'Status': 'Running',
        }]
        self.newCID = MockDockerId()
        self.docker_client.events.return_value = [
            MockEvent(e, docker_id=self.newCID).encoded()
            for e in self.LIFECYCLE]

    def test_remove_onexit(self):
        self.config['RemoveStoppedContainers'] = 'onexit'
        cleaner.run(self.config, self.docker_client)
        self.docker_client.remove_container.assert_called_once_with(
            self.newCID, v=True)

    def test_remove_always(self):
        self.config['RemoveStoppedContainers'] = 'always'
        cleaner.run(self.config, self.docker_client)
        self.docker_client.remove_container.assert_any_call(
            self.existingCID, v=True)
        self.docker_client.remove_container.assert_any_call(
            self.newCID, v=True)
        self.assertEqual(2, self.docker_client.remove_container.call_count)

    def test_remove_never(self):
        self.config['RemoveStoppedContainers'] = 'never'
        cleaner.run(self.config, self.docker_client)
        self.assertEqual(0, self.docker_client.remove_container.call_count)

    def test_container_exited_between_subscribe_events_and_check_existing(self):
        self.config['RemoveStoppedContainers'] = 'always'
        self.docker_client.events.return_value = [
            MockEvent(e, docker_id=self.existingCID).encoded()
            for e in ['die', 'destroy']]
        cleaner.run(self.config, self.docker_client)
        # Subscribed to events before getting the list of existing
        # exited containers?
        self.docker_client.assert_has_calls([
            mock.call.events(since=mock.ANY),
            mock.call.containers(filters={'status': 'exited'})])
        # Asked to delete the container twice?
        self.docker_client.remove_container.assert_has_calls(
            [mock.call(self.existingCID, v=True)] * 2)
        self.assertEqual(2, self.docker_client.remove_container.call_count)
