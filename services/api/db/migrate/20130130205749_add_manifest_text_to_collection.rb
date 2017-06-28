# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddManifestTextToCollection < ActiveRecord::Migration
  def change
    add_column :collections, :manifest_text, :text
  end
end
