# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import itertools
import os
import subprocess
import unittest

import parameterized
from unittest import mock

import arvados
import arvados.util

class KeysetTestHelper:
    def __init__(self, expect, expect_num_retries=0):
        self.n = 0
        self.expect = expect
        self.expect_num_retries = expect_num_retries

    def fn(self, **kwargs):
        assert kwargs == self.expect[self.n][0]
        return self

    def execute(self, num_retries):
        assert num_retries == self.expect_num_retries
        self.n += 1
        return self.expect[self.n-1][1]

_SELECT_FAKE_ITEM = {
    'uuid': 'zzzzz-zyyyz-zzzzzyyyyywwwww',
    'name': 'KeysetListAllTestCase.test_select mock',
    'created_at': '2023-08-28T12:34:56.123456Z',
}

_FAKE_COMPUTED_PERMISSIONS_ITEM = {
    'user_uuid': 'zzzzz-zyyyz-zzzzzyyyyywwwww',
    'target_uuid': 'zzzzz-ttttt-xxxxxyyyyyzzzzz',
    'perm_level': 'can_write',
}

class KeysetListAllTestCase(unittest.TestCase):
    def test_empty(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [])

    def test_oneitem(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", "=", "1"], ["uuid", ">", "1"]]},
            {"items": []}
        ],[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">", "1"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"}])

    def test_onepage2(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "2"], ["uuid", "!=", "2"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}])

    def test_onepage3(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}, {"created_at": "3", "uuid": "3"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "3"], ["uuid", "!=", "3"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}, {"created_at": "3", "uuid": "3"}])


    def test_twopage(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "2"], ["uuid", "!=", "2"]]},
            {"items": [{"created_at": "3", "uuid": "3"}, {"created_at": "4", "uuid": "4"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "4"], ["uuid", "!=", "4"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"},
                              {"created_at": "2", "uuid": "2"},
                              {"created_at": "3", "uuid": "3"},
                              {"created_at": "4", "uuid": "4"}
        ])

    def test_repeated_key(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": []},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}, {"created_at": "2", "uuid": "3"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "2"], ["uuid", "!=", "3"]]},
            {"items": [{"created_at": "2", "uuid": "2"}, {"created_at": "2", "uuid": "4"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", "=", "2"], ["uuid", ">", "4"]]},
            {"items": []}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">", "2"]]},
            {"items": [{"created_at": "3", "uuid": "5"}, {"created_at": "4", "uuid": "6"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "4"], ["uuid", "!=", "6"]]},
            {"items": []}
        ],
        ])

        ls = list(arvados.util.keyset_list_all(ks.fn))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"},
                              {"created_at": "2", "uuid": "2"},
                              {"created_at": "2", "uuid": "3"},
                              {"created_at": "2", "uuid": "4"},
                              {"created_at": "3", "uuid": "5"},
                              {"created_at": "4", "uuid": "6"}
        ])

    def test_onepage_withfilter(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["foo", ">", "bar"]]},
            {"items": [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at asc", "uuid asc"], "filters": [["created_at", ">=", "2"], ["uuid", "!=", "2"], ["foo", ">", "bar"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn, filters=[["foo", ">", "bar"]]))
        self.assertEqual(ls, [{"created_at": "1", "uuid": "1"}, {"created_at": "2", "uuid": "2"}])

    def test_onepage_desc(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["created_at desc", "uuid desc"], "filters": []},
            {"items": [{"created_at": "2", "uuid": "2"}, {"created_at": "1", "uuid": "1"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["created_at desc", "uuid desc"], "filters": [["created_at", "<=", "1"], ["uuid", "!=", "1"]]},
            {"items": []}
        ]])

        ls = list(arvados.util.keyset_list_all(ks.fn, ascending=False))
        self.assertEqual(ls, [{"created_at": "2", "uuid": "2"}, {"created_at": "1", "uuid": "1"}])

    @parameterized.parameterized.expand(
        (fake_item, key_fields, order_key, select)
        for (fake_item, key_fields) in [
            (_SELECT_FAKE_ITEM, ('uuid',)),
            (_FAKE_COMPUTED_PERMISSIONS_ITEM, ('user_uuid', 'target_uuid')),
        ]
        for order_key in fake_item
        if order_key != 'perm_level'
        for count in range(len(fake_item) + 1)
        for select in itertools.combinations(fake_item, count)
    )
    def test_select(self, fake_item, key_fields, order_key, select):
        # keyset_list_all must have both uuid and order_key to function.
        # Test that it selects those fields along with user-specified ones.
        expect_select = {*key_fields, order_key, *select}
        item = {
            key: value
            for key, value in fake_item.items()
            if key in expect_select
        }
        list_func = mock.Mock()
        list_func().execute = mock.Mock(
            side_effect=[
                {'items': [item]},
                {'items': []},
                {'items': []},
            ],
        )
        list_func.reset_mock()
        actual = list(arvados.util.keyset_list_all(list_func, order_key, select=list(select), key_fields=key_fields))
        self.assertEqual(actual, [item])
        calls = list_func.call_args_list
        self.assertTrue(len(calls) >= 2, "list_func() not called enough to exhaust items")
        for args, kwargs in calls:
            self.assertEqual(set(kwargs.get('select', ())), expect_select)
