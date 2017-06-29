# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SeparateRepositoryFromScriptVersion < ActiveRecord::Migration
  include CurrentApiClient

  def fixup pt
    c = pt.components
    c.each do |k, v|
      commit_ish = v["script_version"]
      if commit_ish.andand.index(':')
        want_repo, commit_ish = commit_ish.split(':',2)
        v[:repository] = want_repo
        v[:script_version] = commit_ish
      end
    end
    pt.save!
  end

  def up
    act_as_system_user do
      PipelineTemplate.all.each do |pt|
        fixup pt
      end
      PipelineInstance.all.each do |pt|
        fixup pt
      end
    end
  end

  def down
    raise ActiveRecord::IrreversibleMigration
  end
end
