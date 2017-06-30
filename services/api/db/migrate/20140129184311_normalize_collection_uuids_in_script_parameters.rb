# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class NormalizeCollectionUuidsInScriptParameters < ActiveRecord::Migration
  include CurrentApiClient
  def up
    act_as_system_user do
      PipelineInstance.all.each do |pi|
        pi.save! if fix_values_recursively(pi.components)
      end
      Job.all.each do |j|
        changed = false
        j.script_parameters.each do |p, v|
          if v.is_a? String and v.match /\+K/
            v.gsub! /\+K\@\w+/, ''
            changed = true
          end
        end
        j.save! if changed
      end
    end
  end

  def down
  end

  protected
  def fix_values_recursively fixme
    changed = false
    if fixme.is_a? String
      if fixme.match /\+K/
        fixme.gsub! /\+K\@\w+/, ''
        return true
      else
        return false
      end
    elsif fixme.is_a? Array
      fixme.each do |v|
        changed = fix_values_recursively(v) || changed
      end
    elsif fixme.is_a? Hash
      fixme.each do |p, v|
        changed = fix_values_recursively(v) || changed
      end
    end
    changed
  end
end
