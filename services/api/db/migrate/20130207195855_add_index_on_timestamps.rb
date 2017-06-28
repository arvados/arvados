# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddIndexOnTimestamps < ActiveRecord::Migration
  def tables
    %w{api_clients collections jobs job_steps links logs nodes pipeline_invocations pipelines projects specimens users}
  end

  def change
    tables.each do |t|
      add_index t.to_sym, :created_at
      add_index t.to_sym, :modified_at
    end
  end
end
