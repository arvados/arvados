# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'group_pdhs'

# NOTE: Migration 20190322174136_add_file_info_to_collection.rb
# relies on this test. Change with caution!
class GroupPdhsTest < ActiveSupport::TestCase
  test "pdh_grouping_by_manifest_size" do
    batch_size_max = 200
    pdhs_in = ['x1+30', 'x2+30', 'x3+201', 'x4+100', 'x5+100']
    pdh_lambda = lambda { |last_pdh, &block|
      pdhs = pdhs_in.select{|pdh| pdh > last_pdh} 
      pdhs.each do |p|
        block.call(p)
      end
    }
    batched_pdhs = []
    GroupPdhs.group_pdhs_for_multiple_transactions(pdh_lambda, pdhs_in.size, batch_size_max, "") do |pdhs|
      batched_pdhs << pdhs
    end
    expected = [['x1+30', 'x2+30'], ['x3+201'], ['x4+100', 'x5+100']]
    assert_equal(batched_pdhs, expected)
  end
end
