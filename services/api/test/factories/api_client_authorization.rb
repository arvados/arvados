# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FactoryBot.define do
  factory :api_client_authorization do
    scopes { ['all'] }

    factory :token do
      # Just provides shorthand for "create :api_client_authorization"
    end

    to_create do |instance|
      CurrentApiClientHelper.act_as_user instance.user do
        instance.save!
      end
    end
  end
end
