# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import arvados_fuse
import mock
import unittest
import llfuse
import logging

class InodeTests(unittest.TestCase):

    # The following tests call next(inodes._counter) because inode 1
    # (the root directory) gets special treatment.

    def test_inodes_basic(self):
        cache = arvados_fuse.InodeCache(1000, 4)
        inodes = arvados_fuse.Inodes(cache)
        next(inodes._counter)

        # Check that ent1 gets added to inodes
        ent1 = mock.MagicMock()
        ent1.in_use.return_value = False
        ent1.has_ref.return_value = False
        ent1.persisted.return_value = True
        ent1.objsize.return_value = 500
        inodes.add_entry(ent1)
        self.assertIn(ent1.inode, inodes)
        self.assertIs(inodes[ent1.inode], ent1)
        self.assertEqual(500, cache.total())

    def test_inodes_not_persisted(self):
        cache = arvados_fuse.InodeCache(1000, 4)
        inodes = arvados_fuse.Inodes(cache)
        next(inodes._counter)

        ent1 = mock.MagicMock()
        ent1.in_use.return_value = False
        ent1.has_ref.return_value = False
        ent1.persisted.return_value = True
        ent1.objsize.return_value = 500
        inodes.add_entry(ent1)

        # ent2 is not persisted, so it doesn't
        # affect the cache total
        ent2 = mock.MagicMock()
        ent2.in_use.return_value = False
        ent2.has_ref.return_value = False
        ent2.persisted.return_value = False
        ent2.objsize.return_value = 600
        inodes.add_entry(ent2)
        self.assertEqual(500, cache.total())

    def test_inode_cleared(self):
        cache = arvados_fuse.InodeCache(1000, 4)
        inodes = arvados_fuse.Inodes(cache)
        next(inodes._counter)

        # Check that ent1 gets added to inodes
        ent1 = mock.MagicMock()
        ent1.in_use.return_value = False
        ent1.has_ref.return_value = False
        ent1.persisted.return_value = True
        ent1.objsize.return_value = 500
        inodes.add_entry(ent1)

        # ent3 is persisted, adding it should cause ent1 to get cleared
        ent3 = mock.MagicMock()
        ent3.in_use.return_value = False
        ent3.has_ref.return_value = False
        ent3.persisted.return_value = True
        ent3.objsize.return_value = 600

        self.assertFalse(ent1.clear.called)
        inodes.add_entry(ent3)

        # Won't clear anything because min_entries = 4
        self.assertEqual(2, len(cache._cache_entries))
        self.assertFalse(ent1.clear.called)
        self.assertEqual(1100, cache.total())

        # Change min_entries
        cache.min_entries = 1
        ent1.parent_inode = None
        inodes.cap_cache()
        inodes.wait_remove_queue_empty()
        self.assertEqual(600, cache.total())
        self.assertTrue(ent1.clear.called)

        # Touching ent1 should cause ent3 to get cleared
        ent3.parent_inode = None
        self.assertFalse(ent3.clear.called)
        inodes.inode_cache.update_cache_size(ent1)
        inodes.touch(ent1)
        inodes.wait_remove_queue_empty()
        self.assertTrue(ent3.clear.called)
        self.assertEqual(500, cache.total())

    def test_clear_in_use(self):
        cache = arvados_fuse.InodeCache(1000, 4)
        inodes = arvados_fuse.Inodes(cache)
        next(inodes._counter)

        ent1 = mock.MagicMock()
        ent1.in_use.return_value = True
        ent1.has_ref.return_value = False
        ent1.persisted.return_value = True
        ent1.objsize.return_value = 500
        inodes.add_entry(ent1)

        ent3 = mock.MagicMock()
        ent3.in_use.return_value = False
        ent3.has_ref.return_value = True
        ent3.persisted.return_value = True
        ent3.objsize.return_value = 600
        inodes.add_entry(ent3)

        cache.min_entries = 1

        # ent1, ent3 in use, has ref, can't be cleared
        ent1.clear.called = False
        ent3.clear.called = False
        self.assertFalse(ent1.clear.called)
        self.assertFalse(ent3.clear.called)
        inodes.touch(ent3)
        inodes.wait_remove_queue_empty()
        self.assertFalse(ent1.clear.called)
        self.assertFalse(ent3.clear.called)
        # kernel invalidate gets called anyway
        self.assertTrue(ent3.kernel_invalidate.called)
        self.assertEqual(1100, cache.total())

        # ent1 still in use, ent3 doesn't have ref,
        # so ent3 gets cleared
        ent3.has_ref.return_value = False
        ent1.clear.called = False
        ent3.clear.called = False
        ent3.parent_inode = None
        inodes.touch(ent3)
        inodes.wait_remove_queue_empty()
        self.assertFalse(ent1.clear.called)
        self.assertTrue(ent3.clear.called)
        self.assertEqual(500, cache.total())

    def test_delete(self):
        cache = arvados_fuse.InodeCache(1000, 0)
        inodes = arvados_fuse.Inodes(cache)
        next(inodes._counter)

        ent1 = mock.MagicMock()
        ent1.in_use.return_value = False
        ent1.has_ref.return_value = False
        ent1.persisted.return_value = True
        ent1.objsize.return_value = 500
        inodes.add_entry(ent1)

        ent3 = mock.MagicMock()
        ent3.in_use.return_value = False
        ent3.has_ref.return_value = False
        ent3.persisted.return_value = True
        ent3.objsize.return_value = 600

        # Delete ent1
        self.assertEqual(500, cache.total())
        ent1.ref_count = 0
        with llfuse.lock:
            inodes.del_entry(ent1)
        inodes.wait_remove_queue_empty()
        self.assertEqual(0, cache.total())

        inodes.add_entry(ent3)
        inodes.wait_remove_queue_empty()
        self.assertEqual(600, cache.total())
