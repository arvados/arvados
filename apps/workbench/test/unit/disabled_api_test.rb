# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class DisabledApiTest < ActiveSupport::TestCase
  test 'Job.creatable? is false' do
    use_token(:active) do
      refute(Job.creatable?)
    end
  end
end
