# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FactoryGirl.define do
  factory :api_client do
    is_trusted false
    to_create do |instance|
      CurrentApiClientHelper.act_as_system_user do
        instance.save!
      end
    end
  end
end
