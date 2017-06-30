# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameOrvosToArvados < ActiveRecord::Migration
  def up
    Link.update_all("head_kind=replace(head_kind,'orvos','arvados')")
    Link.update_all("tail_kind=replace(tail_kind,'orvos','arvados')")
    Log.update_all("object_kind=replace(object_kind,'orvos','arvados')")
  end

  def down
    Link.update_all("head_kind=replace(head_kind,'arvados','orvos')")
    Link.update_all("tail_kind=replace(tail_kind,'arvados','orvos')")
    Log.update_all("object_kind=replace(object_kind,'arvados','orvos')")
  end
end
