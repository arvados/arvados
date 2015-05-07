import arvados_fuse
import mock
import unittest

class InodeTests(unittest.TestCase):
    def test_inodes(self):
        cache = arvados_fuse.InodeCache(1000, 4)
        inodes = arvados_fuse.Inodes(cache)

        # Check that ent1 gets added to inodes
        ent1 = mock.MagicMock()
        ent1.return_value.in_use = False
        ent1.persisted.return_value = True
        ent1.clear.return_value = True
        ent1.objsize.return_value = 500
        inodes.add_entry(ent1)
        self.assertIn(ent1.inode, inodes)
        self.assertIs(inodes[ent1.inode], ent1)
        self.assertEqual(500, cache.total())

        # ent2 is not persisted, so it doesn't
        # affect the cache total
        ent2 = mock.MagicMock()
        ent2.return_value.in_use = False
        ent2.persisted.return_value = False
        ent2.objsize.return_value = 600
        inodes.add_entry(ent2)
        self.assertEqual(500, cache.total())

        # ent3 is persisted, adding it should cause ent1 to get cleared
        ent3 = mock.MagicMock()
        ent3.return_value.in_use = False
        ent3.persisted.return_value = True
        ent3.objsize.return_value = 600
        ent3.clear.return_value = True

        self.assertFalse(ent1.clear.called)
        inodes.add_entry(ent3)

        # Won't clear anything because min_entries = 4
        self.assertEqual(2, len(cache._entries))
        self.assertFalse(ent1.clear.called)
        self.assertEqual(1100, cache.total())

        # Change min_entries
        cache.min_entries = 1
        cache.cap_cache()
        self.assertEqual(600, cache.total())
        self.assertTrue(ent1.clear.called)

        # Touching ent1 should cause ent3 to get cleared
        self.assertFalse(ent3.clear.called)
        cache.touch(ent1)
        self.assertTrue(ent3.clear.called)
        self.assertEqual(500, cache.total())

        # ent1, ent3 clear return false, can't be cleared
        ent1.clear.return_value = False
        ent3.clear.return_value = False
        ent1.clear.called = False
        ent3.clear.called = False
        self.assertFalse(ent1.clear.called)
        self.assertFalse(ent3.clear.called)
        cache.touch(ent3)
        self.assertTrue(ent1.clear.called)
        self.assertTrue(ent3.clear.called)
        self.assertEqual(1100, cache.total())

        # ent1 clear return false, so ent3
        # gets cleared
        ent1.clear.return_value = False
        ent3.clear.return_value = True
        ent1.clear.called = False
        ent3.clear.called = False
        self.assertFalse(ent1.clear.called)
        self.assertFalse(ent3.clear.called)
        cache.touch(ent3)
        self.assertTrue(ent1.clear.called)
        self.assertTrue(ent3.clear.called)
        self.assertEqual(500, cache.total())

        # Delete ent1
        ent1.clear.return_value = True
        ent1.ref_count = 0
        inodes.del_entry(ent1)
        self.assertEqual(0, cache.total())
        cache.touch(ent3)
        self.assertEqual(600, cache.total())
