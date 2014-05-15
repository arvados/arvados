#! /usr/bin/env python

import datamanager
import unittest

class TestComputeWeightedReplicationCosts(unittest.TestCase):
  def test_obvious(self):
    self.assertEqual(datamanager.computeWeightedReplicationCosts([1,]),
                     {1:1.0})

  def test_simple(self):
    self.assertEqual(datamanager.computeWeightedReplicationCosts([2,]),
                     {2:2.0})

  def test_even_split(self):
    self.assertEqual(datamanager.computeWeightedReplicationCosts([1,1]),
                     {1:0.5})

  def test_even_split_bigger(self):
    self.assertEqual(datamanager.computeWeightedReplicationCosts([2,2]),
                     {2:1.0})

  def test_uneven_split(self):
    self.assertEqual(datamanager.computeWeightedReplicationCosts([1,2]),
                     {1:0.5, 2:1.5})

  def test_uneven_split_bigger(self):
    self.assertEqual(datamanager.computeWeightedReplicationCosts([1,3]),
                     {1:0.5, 3:2.5})

  def test_uneven_split_jumble(self):
    self.assertEqual(datamanager.computeWeightedReplicationCosts([1,3,6,6,10]),
                     {1:0.2, 3:0.7, 6:1.7, 10:5.7})

  def test_documentation_example(self):
    self.assertEqual(datamanager.computeWeightedReplicationCosts([1,1,3,6,6]),
                     {1:0.2, 3: 0.2 + 2.0 / 3, 6: 0.2 + 2.0 / 3 + 1.5})


if __name__ == '__main__':
  unittest.main()
