# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class PipelineInstanceTest < ActiveSupport::TestCase

  [:has_component_with_no_script_parameters,
   :has_component_with_empty_script_parameters].each do |pi_name|
    test "update pipeline that #{pi_name}" do
      pi = pipeline_instances pi_name

      Thread.current[:user] = users(:active)
      assert_equal PipelineInstance::Ready, pi.state
    end
  end
end
