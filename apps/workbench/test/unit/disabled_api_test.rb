# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class DisabledApiTest < ActiveSupport::TestCase
  test 'Job.creatable? reflects whether jobs.create API is enabled' do
    use_token(:active) do
      assert(Job.creatable?)
    end
    dd = ArvadosApiClient.new_or_current.discovery.deep_dup
    dd[:resources][:jobs][:methods].delete(:create)
    ArvadosApiClient.any_instance.stubs(:discovery).returns(dd)
    use_token(:active) do
      refute(Job.creatable?)
    end
  end
end
