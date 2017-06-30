# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FactoryGirl.define do
  factory :link do
    factory :permission_link do
      link_class 'permission'
    end
  end
end
