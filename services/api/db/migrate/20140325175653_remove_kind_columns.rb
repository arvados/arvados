# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RemoveKindColumns < ActiveRecord::Migration
  include CurrentApiClient

  def up
    remove_column :links, :head_kind
    remove_column :links, :tail_kind
    remove_column :logs, :object_kind
  end

  def down
    add_column :links, :head_kind, :string
    add_column :links, :tail_kind, :string
    add_column :logs, :object_kind, :string

    act_as_system_user do
      Link.all.each do |l|
        l.head_kind = ArvadosModel::resource_class_for_uuid(l.head_uuid).kind if l.head_uuid
        l.tail_kind = ArvadosModel::resource_class_for_uuid(l.tail_uuid).kind if l.tail_uuid
        l.save
      end
      Log.all.each do |l|
        l.object_kind = ArvadosModel::resource_class_for_uuid(l.object_uuid).kind if l.object_uuid
        l.save
      end
    end
  end
end
