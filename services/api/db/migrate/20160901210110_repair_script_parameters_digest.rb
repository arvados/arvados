# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RepairScriptParametersDigest < ActiveRecord::Migration
  def up
    Job.find_each do |j|
      have = j.script_parameters_digest
      want = j.update_script_parameters_digest
      if have != want
        # where().update_all() skips validations, event logging, and
        # timestamp updates, and just runs SQL. (This change is
        # invisible to clients.)
        Job.where('id=?', j.id).update_all(script_parameters_digest: want)
      end
    end
  end

  def down
  end
end
