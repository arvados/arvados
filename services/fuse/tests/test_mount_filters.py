# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import collections
import itertools
import json
import re
import unittest

from pathlib import Path

from parameterized import parameterized

from arvados_fuse import fusedir

from .integration_test import IntegrationTest
from .mount_test_base import MountTestBase
from .run_test_server import fixture

_COLLECTIONS = fixture('collections')
_GROUPS = fixture('groups')
_LINKS = fixture('links')
_USERS = fixture('users')

class DirectoryFiltersTestCase(MountTestBase):
    DEFAULT_ROOT_KWARGS = {
        'enable_write': False,
        'filters': [
            ['collections.name', 'like', 'zzzzz-4zz18-%'],
            # This matches both "A Project" (which we use as the test root)
            # and "A Subproject" (which we assert is found under it).
            ['groups.name', 'like', 'A %roject'],
        ],
    }
    EXPECTED_PATHS = frozenset([
        _COLLECTIONS['foo_collection_in_aproject']['name'],
        _GROUPS['asubproject']['name'],
    ])
    CHECKED_PATHS = EXPECTED_PATHS.union([
        _COLLECTIONS['collection_to_move_around_in_aproject']['name'],
        _GROUPS['subproject_in_active_user_home_project_to_test_unique_key_violation']['name'],
    ])

    @parameterized.expand([
        (fusedir.MagicDirectory, {}, _GROUPS['aproject']['uuid']),
        (fusedir.ProjectDirectory, {'project_object': _GROUPS['aproject']}, '.'),
        (fusedir.SharedDirectory, {'exclude': None}, Path(
            '{first_name} {last_name}'.format_map(_USERS['active']),
            _GROUPS['aproject']['name'],
        )),
    ])
    def test_filtered_path_exists(self, root_class, root_kwargs, subdir):
        root_kwargs = collections.ChainMap(root_kwargs, self.DEFAULT_ROOT_KWARGS)
        self.make_mount(root_class, **root_kwargs)
        dir_path = Path(self.mounttmp, subdir)
        actual = frozenset(
            basename
            for basename in self.CHECKED_PATHS
            if (dir_path / basename).exists()
        )
        self.assertEqual(
            actual,
            self.EXPECTED_PATHS,
            "mount existence checks did not match expected results",
        )

    @parameterized.expand([
        (fusedir.MagicDirectory, {}, _GROUPS['aproject']['uuid']),
        (fusedir.ProjectDirectory, {'project_object': _GROUPS['aproject']}, '.'),
        (fusedir.SharedDirectory, {'exclude': None}, Path(
            '{first_name} {last_name}'.format_map(_USERS['active']),
            _GROUPS['aproject']['name'],
        )),
    ])
    def test_filtered_path_listing(self, root_class, root_kwargs, subdir):
        root_kwargs = collections.ChainMap(root_kwargs, self.DEFAULT_ROOT_KWARGS)
        self.make_mount(root_class, **root_kwargs)
        actual = frozenset(path.name for path in Path(self.mounttmp, subdir).iterdir())
        self.assertEqual(
            actual & self.EXPECTED_PATHS,
            self.EXPECTED_PATHS,
            "mount listing did not include minimum matches",
        )
        extra = frozenset(
            name
            for name in actual
            if not (name.startswith('zzzzz-4zz18-') or name.endswith('roject'))
        )
        self.assertFalse(
            extra,
            "mount listing included results outside filters",
        )


class TagFiltersTestCase(MountTestBase):
    COLL_UUID = _COLLECTIONS['foo_collection_in_aproject']['uuid']
    TAG_NAME = _LINKS['foo_collection_tag']['name']

    @parameterized.expand([
        '=',
        '!=',
    ])
    def test_tag_directory_filters(self, op):
        self.make_mount(
            fusedir.TagDirectory,
            enable_write=False,
            filters=[
                ['links.head_uuid', op, self.COLL_UUID],
            ],
            tag=self.TAG_NAME,
        )
        checked_path = Path(self.mounttmp, self.COLL_UUID)
        self.assertEqual(checked_path.exists(), op == '=')

    @parameterized.expand(itertools.product(
        ['in', 'not in'],
        ['=', '!='],
    ))
    def test_tags_directory_filters(self, coll_op, link_op):
        self.make_mount(
            fusedir.TagsDirectory,
            enable_write=False,
            filters=[
                ['links.head_uuid', coll_op, [self.COLL_UUID]],
                ['links.name', link_op, self.TAG_NAME],
            ],
        )
        if link_op == '!=':
            filtered_path = Path(self.mounttmp, self.TAG_NAME)
        elif coll_op == 'not in':
            # As of 2024-02-09, foo tag only applies to the single collection.
            # If you filter it out via head_uuid, then it disappears completely
            # from the TagsDirectory. Hence we set that tag directory as
            # filtered_path. If any of this changes in the future,
            # it would be fine to append self.COLL_UUID to filtered_path here.
            filtered_path = Path(self.mounttmp, self.TAG_NAME)
        else:
            filtered_path = Path(self.mounttmp, self.TAG_NAME, self.COLL_UUID, 'foo', 'nonexistent')
        expect_path = filtered_path.parent
        self.assertTrue(
            expect_path.exists(),
            f"path not found but should exist: {expect_path}",
        )
        self.assertFalse(
            filtered_path.exists(),
            f"path was found but should be filtered out: {filtered_path}",
        )


class FiltersIntegrationTest(IntegrationTest):
    COLLECTIONS_BY_PROP = {
        coll['properties']['MainFile']: coll
        for coll in _COLLECTIONS.values()
        if coll['owner_uuid'] == _GROUPS['fuse_filters_test_project']['uuid']
    }
    PROP_VALUES = list(COLLECTIONS_BY_PROP)

    for test_n, query in enumerate(['foo', 'ba?']):
        @IntegrationTest.mount([
            '--filters', json.dumps([
                ['collections.properties.MainFile', 'like', query],
            ]),
            '--mount-by-pdh', 'by_pdh',
            '--mount-by-id', 'by_id',
            '--mount-home', 'home',
        ])
        def _test_func(self, query=query):
            pdh_path = Path(self.mnt, 'by_pdh')
            id_path = Path(self.mnt, 'by_id')
            home_path = Path(self.mnt, 'home')
            query_re = re.compile(query.replace('?', '.'))
            for prop_val, coll in self.COLLECTIONS_BY_PROP.items():
                should_exist = query_re.fullmatch(prop_val) is not None
                for path in [
                        pdh_path / coll['portable_data_hash'],
                        id_path / coll['portable_data_hash'],
                        id_path / coll['uuid'],
                        home_path / coll['name'],
                ]:
                    self.assertEqual(
                        path.exists(),
                        should_exist,
                        f"{path} from MainFile={prop_val} exists!={should_exist}",
                    )
        exec(f"test_collection_properties_filters_{test_n} = _test_func")

    for test_n, mount_opts in enumerate([
            ['--home'],
            ['--project', _GROUPS['aproject']['uuid']],
    ]):
        @IntegrationTest.mount([
            '--filters', json.dumps([
                ['collections.name', 'like', 'zzzzz-4zz18-%'],
                ['groups.name', 'like', 'A %roject'],
            ]),
            *mount_opts,
        ])
        def _test_func(self, mount_opts=mount_opts):
            root_path = Path(self.mnt)
            root_depth = len(root_path.parts)
            max_depth = 0
            name_re = re.compile(r'(zzzzz-4zz18-.*|A .*roject)')
            dir_queue = [root_path]
            while dir_queue:
                root_path = dir_queue.pop()
                max_depth = max(max_depth, len(root_path.parts))
                for child in root_path.iterdir():
                    if not child.is_dir():
                        continue
                    match = name_re.fullmatch(child.name)
                    self.assertIsNotNone(
                        match,
                        "found directory with name that should've been filtered",
                    )
                    if not match.group(1).startswith('zzzzz-4zz18-'):
                        dir_queue.append(child)
            self.assertGreaterEqual(
                max_depth,
                root_depth + (2 if mount_opts[0] == '--home' else 1),
                "test descended fewer subdirectories than expected",
            )
        exec(f"test_multiple_name_filters_{test_n} = _test_func")
