# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'record_filters'

class ComputedPermission < ApplicationRecord
  self.table_name = 'materialized_permissions'
  include CurrentApiClient
  include CommonApiTemplate
  extend RecordFilters

  PERM_LEVEL_S = ['none', 'can_read', 'can_write', 'can_manage']

  api_accessible :user do |t|
    t.add :user_uuid
    t.add :target_uuid
    t.add :perm_level_s, as: :perm_level
  end

  protected

  def perm_level_s
    PERM_LEVEL_S[perm_level]
  end

  def self.default_orders
    ["#{table_name}.user_uuid", "#{table_name}.target_uuid"]
  end

  def self.readable_by(*args)
    self
  end

  def self.searchable_columns(operator)
    if !operator.match(/[<=>]/) && !operator.in?(['in', 'not in'])
      []
    else
      ['user_uuid', 'target_uuid']
    end
  end

  def self.limit_index_columns_read
    []
  end

  def self.selectable_attributes
    %w(user_uuid target_uuid perm_level)
  end

  def self.columns_for_attributes(select_attributes)
    select_attributes
  end

  def self.serialized_attributes
    {}
  end
end
