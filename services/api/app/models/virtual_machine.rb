# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class VirtualMachine < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  has_many(:login_permissions,
           -> { where("link_class = 'permission' and name = 'can_login'") },
           foreign_key: 'head_uuid',
           class_name: 'Link',
           primary_key: 'uuid')

  api_accessible :user, extend: :common do |t|
    t.add :hostname
  end

  protected

  def permission_to_create
    current_user and current_user.is_admin
  end
  def permission_to_update
    current_user and current_user.is_admin
  end
end
