# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CurrentApiClientHelper
  extend CurrentApiClient
end

FactoryGirl.define do
  factory :user do
    transient do
      join_groups []
    end
    after :create do |user, evaluator|
      CurrentApiClientHelper.act_as_system_user do
        evaluator.join_groups.each do |g|
          Link.create!(tail_uuid: user.uuid,
                       head_uuid: g.uuid,
                       link_class: 'permission',
                       name: 'can_read')
          Link.create!(tail_uuid: g.uuid,
                       head_uuid: user.uuid,
                       link_class: 'permission',
                       name: 'can_read')
        end
      end
    end
    first_name "Factory"
    last_name "Factory"
    identity_url do
      "https://example.com/#{rand(2**24).to_s(36)}"
    end
    factory :active_user do
      is_active true
      after :create do |user|
        CurrentApiClientHelper.act_as_system_user do
          Link.create!(tail_uuid: user.uuid,
                       head_uuid: Group.where('uuid ~ ?', '-f+$').first.uuid,
                       link_class: 'permission',
                       name: 'can_read')
        end
      end
    end
    to_create do |instance|
      CurrentApiClientHelper.act_as_system_user do
        instance.save!
      end
    end
  end
end
